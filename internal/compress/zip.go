package compress

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
)

// zipWriter implements the http.ResponseWriter interface
// and allows it to transparently compress transmitted data in zip format.
type zipWriter struct {
	w   http.ResponseWriter
	zip *zip.Writer
	buf *bytes.Buffer
}

func NewZipWriter(w http.ResponseWriter) *zipWriter {
	buf := &bytes.Buffer{}
	return &zipWriter{
		w:   w,
		zip: zip.NewWriter(buf),
		buf: buf,
	}
}

func (z *zipWriter) Header() http.Header {
	return z.w.Header()
}

func (z *zipWriter) Write(p []byte) (int, error) {
	writer, err := z.zip.Create("data")
	if err != nil {
		return 0, err
	}
	return writer.Write(p)
}

func (z *zipWriter) WriteHeader(statusCode int) {
	if statusCode < 300 || statusCode == 409 {
		z.w.Header().Set("Content-Encoding", "zip")
	}
	z.w.WriteHeader(statusCode)
}

func (z *zipWriter) Close() error {
	if err := z.zip.Close(); err != nil {
		return err
	}
	_, err := z.w.Write(z.buf.Bytes())
	return err
}

// zipReader implements the io.ReadCloser interface and allows
// to transparently decompress data in zip format.
type zipReader struct {
	r  io.ReadCloser
	zr *zip.Reader
}

func NewZipReader(r io.ReadCloser) (*zipReader, error) {
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return nil, err
	}

	return &zipReader{
		r:  r,
		zr: zr,
	}, nil
}

func (z *zipReader) Read(p []byte) (n int, err error) {
	if len(z.zr.File) == 0 {
		return 0, io.EOF
	}
	file, err := z.zr.File[0].Open()
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return file.Read(p)
}

func (z *zipReader) Close() error {
	return z.r.Close()
}
