package mini_cmux

import (
	"errors"
	"io"
	"net"
	"sync"
)

type Matcher func(io.Reader) bool

var ServerCloseErr = errors.New("server close")

var ConnError = errors.New("conn error")

type MatchWriter func(io.Writer, io.Reader) bool

// New 根据传入的net.listener实例化一个多路复用器
func New(l net.Listener) CMux {
	return &cMux{
		root:   l,
		bufLen: 1024,
		donec:  make(chan struct{}),
	}
}

// CMux 是一个网络连接的多路复用器
type CMux interface {
	// Match 对匹配器进行匹配
	Match(MatchWriter) net.Listener
	// Serve 启动多路复用器
	Serve() error
	// Close 关闭多路复用器
	Close()
}

type matchersListener struct {
	ss MatchWriter
	l  muxListener
}

type cMux struct {
	root   net.Listener
	bufLen int                // 匹配器中缓存连接的队列长度
	sls    []matchersListener // 注册的匹配器列表
	donec  chan struct{}      // 多路复用器关闭channel
	mu     sync.Mutex
}

// Match 对传入的 MatchWriter 进行包装成 muxListener，muxListener实现了 net.Listener 接口
// 用于返回给与匹配器对应的服务端进行连接的获取、处理和关闭等操作
func (m *cMux) Match(matchers MatchWriter) net.Listener {
	ml := muxListener{
		Listener: m.root,
		connc:    make(chan net.Conn, m.bufLen),
		donec:    make(chan struct{}),
	}
	//将该muxListener添加到CMux匹配器列表中
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
			// 关闭各匹配器对应的连接队列
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

// MuxConn 将 net.Conn 包装为 MuxConn 并提供对连接数据的透明嗅探
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

// 开始嗅探
func (m *MuxConn) startSniffing() io.Reader {
	m.buf.reset(true)
	return &m.buf
}

// 结束嗅探
func (m *MuxConn) doneSniffing() {
	m.buf.reset(false)
}
