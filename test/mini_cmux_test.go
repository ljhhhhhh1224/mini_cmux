package test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mini_cmux"
	"mini_cmux/grpcServer"
	hello_grpc "mini_cmux/pb"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	. "github.com/smartystreets/goconvey/convey"
)

type HTTP1Handler struct{}

const (
	HTTP1    = "HTTP1"
	GrpcRESP = "(GRPC)服务端响应SayHi请求"
)

func (*HTTP1Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, HTTP1)
}

func HTTP1Client(errChan chan<- error, addr net.Addr) string {
	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	req, err := http.NewRequest("GET", "http://"+addr.String(), nil)
	if err != nil {
		errChan <- err
	}
	req.Header.Add("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		errChan <- err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			errChan <- err
		}
	}()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errChan <- err
	}
	return string(b)
}

func HTTP1Server(errCh chan<- error, l net.Listener) {
	var mu sync.Mutex
	conns := make(map[net.Conn]struct{})

	defer func() {
		mu.Lock()
		for c := range conns {
			if err := c.Close(); err != nil {
				errCh <- err
			}
		}
		mu.Unlock()
	}()

	s := &http.Server{
		Handler: &HTTP1Handler{},
		ConnState: func(c net.Conn, state http.ConnState) {
			mu.Lock()
			switch state {
			case http.StateNew:
				conns[c] = struct{}{}
			case http.StateClosed:
				delete(conns, c)
			}
			mu.Unlock()
		},
	}
	if err := s.Serve(l); err != nil {
		errCh <- err
	}
}

func TestHTTP1(t *testing.T) {
	Convey("Test HTTP1", t, func() {
		servererrChan := make(chan error)
		clienterrChan := make(chan error)
		l, _ := net.Listen("tcp", "127.0.0.1:0")
		m := mini_cmux.New(l)
		httpl := m.Match(mini_cmux.HTTP1HeaderField("content-type", "application/json"))
		go HTTP1Server(servererrChan, httpl)
		go Serve(servererrChan, m)
		resp := HTTP1Client(clienterrChan, l.Addr())

		So(cap(servererrChan), ShouldEqual, 0)
		So(cap(clienterrChan), ShouldEqual, 0)
		So(resp, ShouldEqual, HTTP1)
	})
}

func gRpcServer(errChan chan<- error, l net.Listener) {
	grpcS := grpc.NewServer()
	hello_grpc.RegisterHelloGRPCServer(grpcS, &grpcServer.Server{})
	err := grpcS.Serve(l)
	if err != nil {
		errChan <- err
	}
}

func gRpcClient(errChan chan<- error, addr string) string {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		errChan <- err
	}
	defer conn.Close()

	client := hello_grpc.NewHelloGRPCClient(conn)
	req, err := client.SayHi(context.Background(), &hello_grpc.Req{Message: "Say hi from grpc client"})
	if err != nil {
		errChan <- err
	}
	return req.GetMessage()
}

func TestGRPC(t *testing.T) {
	Convey("Test GRPC-GO", t, func() {
		servererrChan := make(chan error)
		clienterrChan := make(chan error)
		l, _ := net.Listen("tcp", "127.0.0.1:0")
		m := mini_cmux.New(l)
		grpcl := m.Match(mini_cmux.HTTP2HeaderField("content-type", "application/grpc"))
		go gRpcServer(servererrChan, grpcl)
		go Serve(servererrChan, m)
		resp := gRpcClient(clienterrChan, l.Addr().String())
		So(resp, ShouldEqual, GrpcRESP)
		So(cap(servererrChan), ShouldEqual, 0)
		So(cap(clienterrChan), ShouldEqual, 0)
	})
}

type muxListener struct {
	net.Listener
	connCh chan net.Conn
}

func (l *muxListener) Accept() (net.Conn, error) {
	if c, ok := <-l.connCh; ok {
		return c, nil
	}
	return nil, errors.New("use of closed network connection")
}

func TestBufferReader(t *testing.T) {
	Convey("TestBufferReader", t, func() {
		errCh := make(chan error)
		const str = "bufferReader"
		const times = 5
		writer, reader := net.Pipe()
		go func() {
			if _, err := io.WriteString(writer, strings.Repeat(str, times)); err != nil {
				errCh <- err
			}
			if err := writer.Close(); err != nil {
				errCh <- err
			}
		}()
		ml := &muxListener{
			connCh: make(chan net.Conn, 1),
		}

		defer close(ml.connCh)
		ml.connCh <- reader

		m := mini_cmux.New(ml)

		m.Match(func(w io.Writer, r io.Reader) bool {
			var b [len(str)]byte
			_, _ = r.Read(b[:])
			return false
		})
		anyL := m.Match(mini_cmux.Any())
		go Serve(errCh, m)
		conn, err := anyL.Accept()
		if err != nil {
			errCh <- err
		}

		for i := 0; i < times; i++ {
			var b [len(str)]byte
			n, err := conn.Read(b[:])
			if err != nil {
				errCh <- err
				continue
			}
			So(len(b), ShouldEqual, n)
		}
		So(cap(errCh), ShouldEqual, 0)
	})
}

func TestAny(t *testing.T) {
	Convey("Test Any", t, func() {
		servererrChan := make(chan error)
		clienterrChan := make(chan error)
		l, _ := net.Listen("tcp", "127.0.0.1:0")
		m := mini_cmux.New(l)
		anyl := m.Match(mini_cmux.Any())
		go HTTP1Server(servererrChan, anyl)
		go Serve(servererrChan, m)
		resp := HTTP1Client(clienterrChan, l.Addr())

		So(cap(servererrChan), ShouldEqual, 0)
		So(cap(clienterrChan), ShouldEqual, 0)
		So(resp, ShouldEqual, HTTP1)
	})
}

func TestClose(t *testing.T) {
	errCh := make(chan error)

	l, _ := net.Listen("tcp", "127.0.0.1:0")

	m := mini_cmux.New(l)
	anyl := m.Match(mini_cmux.Any())

	go Serve(errCh, m)

	m.Close()

	if _, err := anyl.Accept(); err != mini_cmux.ServerCloseErr {
		t.Fatal(err)
	}
}

func Serve(errCh chan<- error, muxl mini_cmux.CMux) {
	if err := muxl.Serve(); !strings.Contains(err.Error(), "use of closed") {
		errCh <- err
	}
}
