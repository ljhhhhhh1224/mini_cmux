package mini_cmux

import (
	"errors"
	"io"
	"net"
	"sync"
)

// Matcher matches a connection based on its content.
type Matcher func(io.Reader) bool

var ServerCloseErr = errors.New("server close")

var ConnError = errors.New("conn error")

// MatchWriter is a match that can also write response (say to do handshake).
type MatchWriter func(io.Writer, io.Reader) bool

func New(l net.Listener) CMux {
	return &cMux{
		root:   l,
		bufLen: 1024,
		donec:  make(chan struct{}),
	}
}

// CMux is a multiplexer for network connections.
type CMux interface {
	Match(MatchWriter) net.Listener

	Serve() error

	Close()
}

type matchersListener struct {
	ss MatchWriter
	l  muxListener
}

type cMux struct {
	root   net.Listener
	bufLen int
	sls    []matchersListener
	donec  chan struct{}
	mu     sync.Mutex
}

func (m *cMux) Match(matchers MatchWriter) net.Listener {
	ml := muxListener{
		Listener: m.root,
		connc:    make(chan net.Conn, m.bufLen),
		donec:    make(chan struct{}),
	}
	m.sls = append(m.sls, matchersListener{ss: matchers, l: ml})
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
	// 将 net.Conn 包装为 MuxConn
	muc := newMuxConn(c)

	// 遍历已注册的匹配器列表
	for _, sl := range m.sls {
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
			return nil, ConnError
		}
		return c, nil
	case <-l.donec:
		return nil, ServerCloseErr
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
