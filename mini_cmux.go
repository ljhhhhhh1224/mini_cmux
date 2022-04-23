// Copyright 2016 The CMux Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

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
// Matcher 根据其内容匹配一个连接。
type Matcher func(io.Reader) bool

// MatchWriter is a match that can also write response (say to do handshake).
// MatchWriter 是一个匹配，也可以写响应（比如握手）。
type MatchWriter func(io.Writer, io.Reader) bool

// ErrorHandler handles an error and returns whether
// the mux should continue serving the listener.
// ErrorHandler 处理错误并返回多路复用器是否应该继续为侦听器服务。
type ErrorHandler func(error) bool

var _ net.Error = ErrNotMatched{}

// ErrNotMatched is returned whenever a connection is not matched by any of the matchers registered in the multiplexer.
// 只要连接不被多路复用器中注册的任何匹配器匹配，就会返回 ErrNotMatched。
type ErrNotMatched struct {
	c net.Conn
}

func (e ErrNotMatched) Error() string {
	return fmt.Sprintf("mux: connection %v not matched by an matcher",
		e.c.RemoteAddr())
}

// Temporary implements the net.Error interface.
// 临时实现 net.Error 接口。
func (e ErrNotMatched) Temporary() bool { return true }

// Timeout implements the net.Error interface.
// Timeout 实现 net.Error 接口。
func (e ErrNotMatched) Timeout() bool { return false }

type errListenerClosed string

func (e errListenerClosed) Error() string   { return string(e) }
func (e errListenerClosed) Temporary() bool { return false }
func (e errListenerClosed) Timeout() bool   { return false }

// ErrListenerClosed is returned from muxListener.Accept when the underlying listener is closed.
// 当底层监听器关闭时，从 muxListener.Accept 返回 ErrListenerClosed。
var ErrListenerClosed = errListenerClosed("mux: listener closed")

// ErrServerClosed is returned from muxListener.Accept when mux server is closed.
// 当 mux 服务器关闭时，从 muxListener.Accept 返回 ErrServerClosed。
var ErrServerClosed = errors.New("mux: server closed")

// for readability of readTimeout
// 为了 readTimeout 的可读性
var noTimeout time.Duration

// New instantiates a new connection multiplexer.
// New 实例化一个新的连接多路复用器。
// New 根据传入的网络监听器实例化一个「连接多路复用器」
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
// CMux 是用于网络连接的多路复用器。
type CMux interface {
	// Match returns a net.Listener that sees (i.e., accepts) only
	// the connections matched by at least one of the matcher.
	//
	// The order used to call Match determines the priority of matchers.

	// Match 返回一个 net.Listener 只看到（即接受）
	// 由至少一个匹配器匹配的连接。
	//
	// 调用 Match 的顺序决定了匹配器的优先级。

	Match(Matcher) net.Listener
	// MatchWithWriters returns a net.Listener that accepts only the
	// connections that matched by at least of the matcher writers.
	//
	// Prefer Matchers over MatchWriters, since the latter can write on the
	// connection before the actual handler.
	//
	// The order used to call Match determines the priority of matchers.

	// MatchWithWriters 返回一个 net.Listener，它只接受
	// 至少由匹配器编写器匹配的连接。
	//
	// 比 MatchWriters 更喜欢 Matchers，因为后者可以写在
	// 实际处理程序之前的连接。
	//
	// 调用 Match 的顺序决定了匹配器的优先级。
	//MatchWithWriters(MatchWriter) net.Listener
	// Serve starts multiplexing the listener. Serve blocks and perhaps
	// should be invoked concurrently within a go routine.

	// 服务开始多路复用监听器。 服务块，也许
	// 应该在 go 例程中同时调用。
	Serve() error

	// Closes cmux server and stops accepting any connections on listener
	// 关闭 cmux 服务器并停止接受侦听器上的任何连接
	Close()

	// HandleError registers an error handler that handles listener errors.
	// HandleError 注册一个处理监听器错误的错误处理器。
	HandleError(ErrorHandler)

	// sets a timeout for the read of matchers
	// 设置读取匹配器的超时时间
	SetReadTimeout(time.Duration)
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
//这里是对一个listen 赋予多个类型(grpc,http1.0或2.0的网络监听)
//func matchersToMatchWriters(matcher Matcher) MatchWriter {
//		cm := matcher
//		mws := func(w io.Writer, r io.Reader) bool {
//			return cm(r)
//		}
//	return mws
//}

// Match 简化函数，会对 Matcher 进行转换为 MatchWriter 并调用 MatchWithWriters
//func (m *cMux) Match(matchers Matcher) net.Listener {
//	mws := matchersToMatchWriters(matchers)
//	return m.MatchWithWriters(mws)
//}

// MatchWithWriters 为传入的匹配器列表初始化一个网络连接包装器，该包装器实现了 net.Listener 接口 用于返回给与匹配器对应的服务端进行连接的 「获取」、「处理」、「关闭」等操作
//func (m *cMux) MatchWithWriters(matchers MatchWriter) net.Listener {
//	ml := muxListener{
//		Listener: m.root,
//		connc:    make(chan net.Conn, m.bufLen),
//		donec:    make(chan struct{}),
//	}
//	// 将传入的匹配器列表打包到 cmux 中
//	m.sls = append(m.sls, matchersListener{ss: matchers, l: ml})
//	return ml
//}

func (m *cMux) Match(matchers Matcher) net.Listener {
	cm := matchers
	mws := func(w io.Writer, r io.Reader) bool {
		return cm(r)
	}
	ml := muxListener{
		Listener: m.root,
		connc:    make(chan net.Conn, m.bufLen),
		donec:    make(chan struct{}),
	}
	// 将传入的匹配器列表打包到 cmux 中
	m.sls = append(m.sls, matchersListener{ss: mws, l: ml})
	return ml
}

func (m *cMux) SetReadTimeout(t time.Duration) {
	m.readTimeout = t
}

func (m *cMux) Serve() error {
	var wg sync.WaitGroup

	//关闭 cmux 实例，并调用所有已注册的匹配器 Close
	defer func() {
		m.closeDoneChans()
		wg.Wait()

		for _, sl := range m.sls {
			close(sl.l.connc)
			// 关闭连接
			// Drain the connections enqueued for the listener.
			for c := range sl.l.connc {
				_ = c.Close()
			}
		}
	}()

	for {
		c, err := m.root.Accept()
		// handle error
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
	//设置超时时间
	if m.readTimeout > noTimeout {
		_ = c.SetReadDeadline(time.Now().Add(m.readTimeout))
	}
	// 遍历已注册的匹配器列表
	for _, sl := range m.sls {
			//遍历MatchWriter,s 为matchwriter
			// 根据连接的内容返回匹配结果，如匹配且 io.Writer 非空则
			// 对 muc.Conn 进行写入
			matched := sl.ss(muc.Conn, muc.startSniffing())
			if matched {
				muc.doneSniffing()
				if m.readTimeout > noTimeout {
					_ = c.SetReadDeadline(time.Time{})
				}
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
// MuxConn 包装一个 net.Conn 并提供对连接数据的透明嗅探。
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

// 当Read遇到错误或文件结束条件后
// 成功读取 n > 0 个字节，它返回的个数
// 读取的字节数。 它可能会从同一个调用返回（非零）错误
// 或者从后续调用中返回错误（并且 n == 0）。
// 这种一般情况的一个实例是 Reader 返回
// 输入流末尾的非零字节数可能
// 返回 err == EOF 或 err == nil。 下一次阅读应该
// 返回 0，EOF。
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
