package artifact

import (
	"hash"
	"io"
)

// hashingReader sums the bytes read using a pluggable hasher implementation.
type hashingReader struct {
	hasher hash.Hash
	reader io.Reader
}

func newHashingReader(hasher hash.Hash, reader io.Reader) *hashingReader {
	return &hashingReader{
		hasher: hasher,
		reader: reader,
	}
}

func (s *hashingReader) Read(p []byte) (int, error) {
	n, err := s.reader.Read(p)
	if n > 0 {
		ni, erri := s.hasher.Write(p[:n])
		if ni != n {
			panic("Hasher didn't process all bytes")
		}
		if erri != nil {
			panic("Hasher returned error")
		}
	}
	return n, err
}
