package compress

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"strings"
)

// TarReader implements io.ReadCloser for reading the content of a CSV file from a TAR archive.
type TarReader struct {
	current io.Reader
	tr      *tar.Reader
	eof     bool
}

// NewTarReader creates a new TarReader, extracting the first found CSV file from the TAR archive.
func NewTarReader(r io.ReadCloser) (*TarReader, error) {
	defer r.Close()

	// Read the entire archive into a buffer
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}

	// Create a tar.Reader
	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))

	// Search for the first CSV file
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeReg && strings.HasSuffix(strings.ToLower(header.Name), ".csv") {
			return &TarReader{
				current: tr,
				tr:      tr,
				eof:     false,
			}, nil
		}
	}

	return nil, errors.New("CSV file not found in the TAR archive")
}

// Read reads data from the current CSV file.
func (t *TarReader) Read(p []byte) (int, error) {
	if t.eof {
		return 0, io.EOF
	}
	n, err := t.current.Read(p)
	if err == io.EOF {
		t.eof = true
	}
	return n, err
}

// Close finalizes the reading process.
func (t *TarReader) Close() error {
	return nil // Required to satisfy the io.ReadCloser interface
}
