package util

import (
	"io"
	"net/http"
)

type FakeCloser struct {
	io.Reader
}

func NewFakeCloser(r io.Reader) *FakeCloser {
	return &FakeCloser{Reader: r}
}

func (f *FakeCloser) Close() error {
	return nil
}

type FlushingWriter struct {
	w io.Writer
	f http.Flusher
}

func NewFlushingWriter(w io.Writer, flusher http.Flusher) *FlushingWriter {
	return &FlushingWriter{
		w: w,
		f: flusher,
	}
}

func (w *FlushingWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.f.Flush()
	return n, err
}

type MultiReaderCloser struct {
	io.Writer
	Writers []io.WriteCloser
}

func (w *MultiReaderCloser) Close() error {
	for _, w := range w.Writers {
		w.Close()
	}
	return nil
}
