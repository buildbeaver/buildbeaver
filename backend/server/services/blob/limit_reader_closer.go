package blob

import "io"

type LimitReaderCloser struct {
	rc io.ReadCloser
	lr io.Reader
}

func NewLimitReaderCloser(rc io.ReadCloser, n int64) *LimitReaderCloser {
	return &LimitReaderCloser{
		rc: rc,
		lr: io.LimitReader(rc, n),
	}
}

func (l *LimitReaderCloser) Read(p []byte) (n int, err error) {
	return l.lr.Read(p)
}

func (l *LimitReaderCloser) Close() error {
	return l.rc.Close()
}
