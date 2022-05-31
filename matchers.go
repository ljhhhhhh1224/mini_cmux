package mini_cmux

import (
	"bufio"
	"io"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

// HTTP1HeaderField 返回一个匹配 HTTP 1 连接的第一个请求的头字段的匹配器。

func HTTP1HeaderField(name, value string) MatchWriter {
	return func(w io.Writer, r io.Reader) bool {
		req, err := http.ReadRequest(bufio.NewReader(r))
		if err != nil {
			return false
		}
		return req.Header.Get(name) == value
	}
}

func HTTP2HeaderField(name, value string) MatchWriter {
	return func(w io.Writer, r io.Reader) bool {
		return matchHTTP2Field(w, r, name, func(gotValue string) bool {
			return gotValue == value
		})
	}
}

func matchHTTP2Field(w io.Writer, r io.Reader, name string, matches func(string) bool) (matched bool) {
	if !hasHTTP2Preface(r) {
		return false
	}

	done := false
	framer := http2.NewFramer(w, r)
	hdec := hpack.NewDecoder(uint32(4<<10), func(hf hpack.HeaderField) {
		if hf.Name == name {
			done = true
			if matches(hf.Value) {
				matched = true
			}
		}
	})
	for {
		f, err := framer.ReadFrame()
		if err != nil {
			return false
		}

		switch f := f.(type) {
		case *http2.SettingsFrame:
			// Sender acknoweldged the SETTINGS frame. No need to write
			// SETTINGS again.
			if f.IsAck() {
				break
			}
			if err := framer.WriteSettings(); err != nil {
				return false
			}
		case *http2.ContinuationFrame:
			if _, err := hdec.Write(f.HeaderBlockFragment()); err != nil {
				return false
			}
			done = done || f.FrameHeader.Flags&http2.FlagHeadersEndHeaders != 0
		case *http2.HeadersFrame:
			if _, err := hdec.Write(f.HeaderBlockFragment()); err != nil {
				return false
			}
			done = done || f.FrameHeader.Flags&http2.FlagHeadersEndHeaders != 0
		}

		if done {
			return matched
		}
	}
}

//检查读取的HTTP2 preface
func hasHTTP2Preface(r io.Reader) bool {
	var b [len(http2.ClientPreface)]byte
	last := 0

	for {
		n, err := r.Read(b[last:])
		if err != nil {
			return false
		}

		last += n
		eq := string(b[:last]) == http2.ClientPreface[:last]
		if last == len(http2.ClientPreface) {
			return eq
		}
		if !eq {
			return false
		}
	}
}
