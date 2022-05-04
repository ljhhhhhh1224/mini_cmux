package mini_cmux

import (
	"errors"
	"io"
	"net"
	"sync"
)

// Matcher matches a connection based on its content.
type Matcher func(io.Reader) bool

// MatchWriter is a match that can also write response (say to do handshake).
type MatchWriter func(io.Writer, io.Reader) bool


// ErrServerClosed is returned from muxListener.Accept when mux server is closed.
var ErrServerClosed = errors.New("mux: server closed")

var ErrListenerClosed = errors.New("mux: listener closed")



func New(l net.Listener) CMux {
	return &cMux{
		root:        l,
		bufLen:      1024,
		donec:       make(chan struct{}),
	}
}

// CMux is a multiplexer for network connections.
type CMux interface {

	Match(Matcher) net.Listener

	Serve() error

	Close()

}

type matchersListener struct {
	ss MatchWriter
	l  muxListener
}

type cMux struct {
	root        net.Listener
	bufLen      int
	sls         []matchersListener
	donec       chan struct{}
	mu          sync.Mutex
}



func (m *cMux) Match(matchers Matcher) net.Listener {
	matcherWriter := func(w io.Writer, r io.Reader) bool {
		return matchers(r)
	}
	ml := muxListener{
		Listener: m.root,
		connc:    make(chan net.Conn, m.bufLen),
		donec:    make(chan struct{}),
	}
	m.sls = append(m.sls, matchersListener{ss: matcherWriter, l: ml})
	return ml
}

func (m *cMux) Serve() error {
	var wg sync.WaitGroup

	defer func() {
		m.closeDoneChans()
		wg.Wait()

		for _, sl := range m.sls {
			close(sl.l.connc)
			// Drain the connections enqueued for the listener.
			for c := range sl.l.connc {
				_ = c.Close()
			}
		}
	}()

	for {
		c, err := m.root.Accept()
		if err != nil {
			return err
		}

		wg.Add(1)
		go m.serve(c, m.donec, &wg)
	}
}

func (m *cMux) serve(c net.Conn, donec <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	// 将 net.Conn 包装为 MuxConn 并提供对连接数据的透明嗅探
	muc := newMuxConn(c)

	// 遍历已注册的匹配器列表
	for _, sl := range m.sls {
		//遍历MatchWriter

		// 根据连接的内容返回匹配结果，如匹配且 io.Writer 非空则对 muc.Conn 进行写入
		// 下面的ss为
		// matcherWriter := func(w io.Writer, r io.Reader) bool {
		//  return matchers(r)
		// }
		matched := sl.ss(muc.Conn, muc.startSniffing())
		if matched {
			muc.doneSniffing()
			select {
			// 将匹配成功的连接放入匹配器的缓存队列中，结束
			case sl.l.connc <- muc:
				// 如果多路复用器标识为终止，则关闭连接，结束
			case <-donec:
				_ = c.Close()
			}
			return
		}
	}
	// 如果执行到这里，意味这个连接没有被任何已注册的匹配器所匹配成功
	// 这里会将 ErrNotMatched 这个错误交给多路复用器的 「错误处理函数」
	c.Close()

}

func (m *cMux) Close() {
	m.closeDoneChans()
}

func (m *cMux) closeDoneChans() {
	m.mu.Lock()
	defer m.mu.Unlock()

	select {
	case <-m.donec:
		// Already closed. Don't close again
	default:
		close(m.donec)
	}
	for _, sl := range m.sls {
		select {
		case <-sl.l.donec:
			// Already closed. Don't close again
		default:
			close(sl.l.donec)
		}
	}
}

type muxListener struct {
	net.Listener
	connc chan net.Conn
	donec chan struct{}
}

func (l muxListener) Accept() (net.Conn, error) {
	select {
	case c, ok := <-l.connc:
		if !ok {
			return nil,ErrListenerClosed
		}
		return c, nil
	case <-l.donec:
		return nil, ErrServerClosed
	}
}

// MuxConn wraps a net.Conn and provides transparent sniffing of connection data.
type MuxConn struct {
	net.Conn
	buf bufferedReader
}

func newMuxConn(c net.Conn) *MuxConn {
	return &MuxConn{
		Conn: c,
		buf:  bufferedReader{source: c},
	}
}


func (m *MuxConn) Read(p []byte) (int, error) {
	return m.buf.Read(p)
}

func (m *MuxConn) startSniffing() io.Reader {
	m.buf.reset(true)
	return &m.buf
}

func (m *MuxConn) doneSniffing() {
	m.buf.reset(false)
}