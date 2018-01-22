package interpol

import (
	"bytes"
	"io"
)

type errRWFunc func(b []byte) (int, error)

type errRW struct {
	xn  int
	n   int
	err error
}

func (rw *errRW) do(f errRWFunc, b []byte) (int, error) {
	var tn int
	for i := 0; i < len(b); i++ {
		if rw.n >= rw.xn {
			return tn, rw.err
		}
		n, err := f(b[i : i+1])
		if err != nil {
			return tn, err
		}
		tn += n
		rw.n += n
	}
	return tn, nil
}

type errWriter struct {
	errRW
	buf *bytes.Buffer
}

func newErrWriter() *errWriter {
	return &errWriter{buf: bytes.NewBuffer(nil)}
}

func (w *errWriter) Write(b []byte) (int, error) {
	return w.do(w.buf.Write, b)
}

type errReader struct {
	errRW
	r io.Reader
}

func newErrReader(r io.Reader) *errReader {
	return &errReader{r: r}
}

func (r *errReader) Read(b []byte) (int, error) {
	return r.do(r.r.Read, b)
}
