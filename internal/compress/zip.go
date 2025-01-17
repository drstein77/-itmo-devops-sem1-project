package compress

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"strings"
)

// ZipReader implements io.ReadCloser for reading the content of a CSV file from a ZIP archive.
type ZipReader struct {
	current io.ReadCloser
}

// NewZipReader creates a new ZipReader, extracting the first found CSV file from the ZIP archive.
func NewZipReader(r io.ReadCloser) (*ZipReader, error) {
	defer r.Close()

	// Read the entire archive into a buffer
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}

	// Create a zip.Reader
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return nil, err
	}

	// Search for the first CSV file
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			return &ZipReader{current: rc}, nil
		}
	}

	return nil, errors.New("CSV file not found in the ZIP archive")
}

// Read reads data from the current CSV file.
func (z *ZipReader) Read(p []byte) (int, error) {
	return z.current.Read(p)
}

// Close closes the current CSV file.
func (z *ZipReader) Close() error {
	return z.current.Close()
}

// ZipWriter implements packaging data into a ZIP archive.
type ZipWriter struct {
	zipWriter *zip.Writer
	file      io.Writer
}

// NewZipWriter creates a new ZipWriter with the specified file name inside the archive.
func NewZipWriter(w io.Writer, fileName string) (*ZipWriter, error) {
	zw := zip.NewWriter(w)
	f, err := zw.Create(fileName)
	if err != nil {
		return nil, err
	}
	return &ZipWriter{
		zipWriter: zw,
		file:      f,
	}, nil
}

// Write writes data to a file inside the ZIP archive.
func (z *ZipWriter) Write(p []byte) (int, error) {
	return z.file.Write(p)
}

// Close closes the ZIP archive.
func (z *ZipWriter) Close() error {
	return z.zipWriter.Close()
}
