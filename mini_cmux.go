package mini_cmux
import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Matcher matches a connection based on its content.
type Matcher func(io.Reader) bool

// MatchWriter is a match that can also write response (say to do handshake).
type MatchWriter func(io.Writer, io.Reader) bool

// ErrorHandler handles an error and returns whether
// the mux should continue serving the listener.
type ErrorHandler func(error) bool

var _ net.Error = ErrNotMatched{}

// ErrNotMatched is returned whenever a connection is not matched by any of
// the matchers registered in the multiplexer.
type ErrNotMatched struct {
	c net.Conn
}

func (e ErrNotMatched) Error() string {
	return fmt.Sprintf("mux: connection %v not matched by an matcher",
		e.c.RemoteAddr())
}

// Temporary implements the net.Error interface.
func (e ErrNotMatched) Temporary() bool { return true }

// Timeout implements the net.Error interface.
func (e ErrNotMatched) Timeout() bool { return false }

type errListenerClosed string

func (e errListenerClosed) Error() string   { return string(e) }
func (e errListenerClosed) Temporary() bool { return false }
func (e errListenerClosed) Timeout() bool   { return false }

// ErrListenerClosed is returned from muxListener.Accept when the underlying
// listener is closed.
var ErrListenerClosed = errListenerClosed("mux: listener closed")

// ErrServerClosed is returned from muxListener.Accept when mux server is closed.
var ErrServerClosed = errors.New("mux: server closed")

// for readability of readTimeout
var noTimeout time.Duration

// New instantiates a new connection multiplexer.
func New(l net.Listener) CMux {
	return &cMux{
		root:        l,
		bufLen:      1024,
		errh:        func(_ error) bool { return true },
		donec:       make(chan struct{}),
		readTimeout: noTimeout,
	}
}

// CMux is a multiplexer for network connections.
type CMux interface {
	// Match returns a net.Listener that sees (i.e., accepts) only
	// the connections matched by at least one of the matcher.
	//
	// The order used to call Match determines the priority of matchers.
	Match(Matcher) net.Listener
	// MatchWithWriters returns a net.Listener that accepts only the
	// connections that matched by at least of the matcher writers.
	//
	// Prefer Matchers over MatchWriters, since the latter can write on the
	// connection before the actual handler.
	//
	// The order used to call Match determines the priority of matchers.
	// Serve starts multiplexing the listener. Serve blocks and perhaps
	// should be invoked concurrently within a go routine.
	Serve() error
	// Closes cmux server and stops accepting any connections on listener
	Close()
	// HandleError registers an error handler that handles listener errors.
	HandleError(ErrorHandler)
}

type matchersListener struct {
	ss MatchWriter
	l  muxListener
}

type cMux struct {
	root        net.Listener
	bufLen      int
	errh        ErrorHandler
	sls         []matchersListener
	readTimeout time.Duration
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


func (m *cMux) SetReadTimeout(t time.Duration) {
	m.readTimeout = t
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
			if !m.handleErr(err) {
				return err
			}
			continue
		}

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
	// _ = c.SetReadDeadline(time.Now().Add(m.readTimeout))
	//}
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
			//这里暂时不进行超时处理
			//if m.readTimeout > noTimeout {
			// _ = c.SetReadDeadline(time.Time{})
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
	err := ErrNotMatched{c: c}
	if !m.handleErr(err) {
		_ = m.root.Close()
	}
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

func (m *cMux) HandleError(h ErrorHandler) {
	m.errh = h
}

func (m *cMux) handleErr(err error) bool {
	if !m.errh(err) {
		return false
	}

	if ne, ok := err.(net.Error); ok {
		return ne.Temporary()
	}

	return false
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
			return nil, ErrListenerClosed
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

// From the io.Reader documentation:
//
// When Read encounters an error or end-of-file condition after
// successfully reading n > 0 bytes, it returns the number of
// bytes read.  It may return the (non-nil) error from the same call
// or return the error (and n == 0) from a subsequent call.
// An instance of this general case is that a Reader returning
// a non-zero number of bytes at the end of the input stream may
// return either err == EOF or err == nil.  The next Read should
// return 0, EOF.
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