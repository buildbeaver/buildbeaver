package util

import (
	"io"
)

// CountingReader counts the number of bytes read from reader.
type CountingReader struct {
	reader io.Reader
	count  uint64
}

func NewCountingReader(reader io.Reader) *CountingReader {
	return &CountingReader{
		reader: reader,
	}
}

func (s *CountingReader) Read(p []byte) (int, error) {
	n, err := s.reader.Read(p)
	s.count += uint64(n)
	return n, err
}

func (s *CountingReader) Count() uint64 {
	return s.count
}
