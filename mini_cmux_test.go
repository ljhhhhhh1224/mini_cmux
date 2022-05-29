package mini_cmux

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

type HTTP1Handler struct{}

func (*HTTP1Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "HTTP1")
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
	Convey("Test HTTP1 Handler", t, func() {
		servererrChan := make(chan error)
		clienterrChan := make(chan error)
		l, _ := net.Listen("tcp", "127.0.0.1:0")
		m := New(l)
		httpl := m.Match(HTTP1HeaderField("content-type", "application/json"))
		go HTTP1Server(servererrChan, httpl)
		go safeServe(servererrChan, m)
		resp := HTTP1Client(clienterrChan, l.Addr())

		So(cap(servererrChan), ShouldEqual, 0)
		So(cap(clienterrChan), ShouldEqual, 0)
		So(resp, ShouldEqual, "HTTP1")
	})
}

func safeDial(t *testing.T, addr net.Addr) (*rpc.Client, func()) {
	c, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		t.Fatal(err)
	}
	return c, func() {
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func safeServe(errCh chan<- error, muxl CMux) {
	if err := muxl.Serve(); !strings.Contains(err.Error(), "use of closed") {
		errCh <- err
	}
}

func leakCheck(t testing.TB) func() {
	orig := map[string]bool{}
	for _, g := range interestingGoroutines() {
		orig[g] = true
	}
	return func() {
		// Loop, waiting for goroutines to shut down.
		// Wait up to 5 seconds, but finish as quickly as possible.
		deadline := time.Now().Add(5 * time.Second)
		for {
			var leaked []string
			for _, g := range interestingGoroutines() {
				if !orig[g] {
					leaked = append(leaked, g)
				}
			}
			if len(leaked) == 0 {
				return
			}
			if time.Now().Before(deadline) {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			for _, g := range leaked {
				t.Errorf("Leaked goroutine: %v", g)
			}
			return
		}
	}
}

func interestingGoroutines() (gs []string) {
	buf := make([]byte, 2<<20)
	buf = buf[:runtime.Stack(buf, true)]
	for _, g := range strings.Split(string(buf), "\n\n") {
		sl := strings.SplitN(g, "\n", 2)
		if len(sl) != 2 {
			continue
		}
		stack := strings.TrimSpace(sl[1])
		if strings.HasPrefix(stack, "testing.RunTests") {
			continue
		}

		if stack == "" ||
			strings.Contains(stack, "main.main()") ||
			strings.Contains(stack, "testing.Main(") ||
			strings.Contains(stack, "runtime.goexit") ||
			strings.Contains(stack, "created by runtime.gc") ||
			strings.Contains(stack, "interestingGoroutines") ||
			strings.Contains(stack, "runtime.MHeap_Scavenger") {
			continue
		}
		gs = append(gs, g)
	}
	sort.Strings(gs)
	return
}
