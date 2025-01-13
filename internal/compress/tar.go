package compress

import (
	"archive/tar"
	"bytes"
	"io"
	"net/http"
)

// tarWriter implements the http.ResponseWriter interface
// and allows it to transparently compress transmitted data in tar format.
type tarWriter struct {
	w   http.ResponseWriter
	tar *tar.Writer
	buf *bytes.Buffer
}

func NewTarWriter(w http.ResponseWriter) *tarWriter {
	buf := &bytes.Buffer{}
	return &tarWriter{
		w:   w,
		tar: tar.NewWriter(buf),
		buf: buf,
	}
}

func (t *tarWriter) Header() http.Header {
	return t.w.Header()
}

func (t *tarWriter) Write(p []byte) (int, error) {
	hdr := &tar.Header{
		Name: "data",
		Size: int64(len(p)),
	}
	if err := t.tar.WriteHeader(hdr); err != nil {
		return 0, err
	}
	return t.tar.Write(p)
}

func (t *tarWriter) WriteHeader(statusCode int) {
	if statusCode < 300 || statusCode == 409 {
		t.w.Header().Set("Content-Encoding", "tar")
	}
	t.w.WriteHeader(statusCode)
}

func (t *tarWriter) Close() error {
	if err := t.tar.Close(); err != nil {
		return err
	}
	_, err := t.w.Write(t.buf.Bytes())
	return err
}

// tarReader implements the io.ReadCloser interface and allows
// to transparently decompress data in tar format.
type tarReader struct {
	r   io.ReadCloser
	tar *tar.Reader
}

func NewTarReader(r io.ReadCloser) (*tarReader, error) {
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}

	return &tarReader{
		r:   r,
		tar: tar.NewReader(bytes.NewReader(buf.Bytes())),
	}, nil
}

func (t *tarReader) Read(p []byte) (n int, err error) {
	_, err = t.tar.Next()
	if err != nil {
		return 0, err
	}
	return t.tar.Read(p)
}

func (t *tarReader) Close() error {
	return t.r.Close()
}
