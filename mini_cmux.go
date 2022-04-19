package mini_cmux

import (
	"io"
	"net"
	"sync"
)

// Matcher 根据内容匹配连接
type Matcher func(io.Reader) bool

// MatchWriter 匹配或响应一个链接
type MatchWriter func(io.Writer, io.Reader) bool

// CMux 定义CMux接口，这里先定义基本功能
type CMux interface {
	// Match 匹配器匹配连接
	Match(...Matcher) net.Listener
	// Serve 多路复用器开始服务
	Serve() error
	// Close 多路复用器关闭服务
	Close()
}

type muxListener struct {
	net.Listener
	connc chan net.Conn
	donec chan struct{}
}

type matchersListener struct {
	ss MatchWriter
	l  muxListener
}

//cMux 定义cMux结构体
type cMux struct {
	root        net.Listener
	bufLen      int
	//errh        ErrorHandler
	sls         []matchersListener
	//readTimeout time.Duration
	donec       chan struct{}
	mu          sync.Mutex
}
//
//func New(l net.Listener) CMux {
//	return &cMux{
//		root:        l,
//		bufLen:      1024,
//		donec:       make(chan struct{}),
//	}
//}


// Match 将matcher 和 net.listen进行绑定
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

	//关闭 cmux 实例，并调用所有已注册的匹配器 Close
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
		c,_ := m.root.Accept()
		// handle error,这里暂时不进行异常处理
		//if err != nil {
		//	if !m.handleErr(err) {
		//		return err
		//	}
		//	continue
		//}

		wg.Add(1)
		go m.serve(c, m.donec, &wg)
	}
}

func (m *cMux) serve(c net.Conn, donec <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	// 将 net.Conn 包装为 MuxConn 并提供对连接数据的透明嗅探
	muc := newMuxConn(c)
	//设置超时时间,目前不需要这个功能
	//if m.readTimeout > noTimeout {
	//	_ = c.SetReadDeadline(time.Now().Add(m.readTimeout))
	//}
	// 遍历已注册的匹配器列表
	for _, sl := range m.sls {
		//遍历MatchWriter

			// 根据连接的内容返回匹配结果，如匹配且 io.Writer 非空则对 muc.Conn 进行写入
			// 下面的ss为
			//	matcherWriter := func(w io.Writer, r io.Reader) bool {
			//		return matchers(r)
			//	}
			matched := sl.ss(muc.Conn, muc.startSniffing())
			if matched {
				muc.doneSniffing()
				//这里暂时不进行超时处理
				//if m.readTimeout > noTimeout {
				//	_ = c.SetReadDeadline(time.Time{})
				//}
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
	_ = c.Close()
	//后续进行处理
	//err := ErrNotMatched{c: c}
	//if !m.handleErr(err) {
	//	_ = m.root.Close()
	//}
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
