package mini_cmux

import (
	"bytes"
	"io"
)

// bufferedReader 实现了 io.Reader 接口,
type bufferedReader struct {
	source     io.Reader
	buffer     bytes.Buffer
	bufferRead int  //已读字节数
	bufferSize int  //总字节数
	sniffing   bool //状态
	lastErr    error
}

func (s *bufferedReader) Read(p []byte) (int, error) {

	if s.bufferSize > s.bufferRead {
		// If we have already read something from the buffer before, we return the
		// same data and the last error if any. We need to immediately return,
		// otherwise we may block for ever, if we try to be smart and call
		// source.Read() seeking a little bit of more data.
		// 在此之前在buffer读取过数据的话,继续从buffer中读取
		bn := copy(p, s.buffer.Bytes()[s.bufferRead:s.bufferSize])
		s.bufferRead += bn
		return bn, s.lastErr
	} else if !s.sniffing && s.buffer.Cap() != 0 {
		// 如果嗅探结束就重置buffer
		s.buffer = bytes.Buffer{}
	}

	//如果在buffer中没有读取到数据，则从source中读取

	sn, sErr := s.source.Read(p)
	if sn > 0 && s.sniffing {
		s.lastErr = sErr
		if wn, wErr := s.buffer.Write(p[:sn]); wErr != nil {
			return wn, wErr
		}
	}
	return sn, sErr
}

//reset 初始化bufferedReader
func (s *bufferedReader) reset(snif bool) {
	s.sniffing = snif
	s.bufferRead = 0
	s.bufferSize = s.buffer.Len()
}
