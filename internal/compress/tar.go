package compress

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"strings"
)

// TarReader реализует io.ReadCloser для чтения содержимого CSV файла из TAR архива.
type TarReader struct {
	current io.Reader
	tr      *tar.Reader
	eof     bool
}

// NewTarReader создает новый TarReader, извлекая первый найденный CSV файл из TAR архива.
func NewTarReader(r io.ReadCloser) (*TarReader, error) {
	defer r.Close()

	// Читаем весь архив в буфер
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}

	// Создаем tar.Reader
	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))

	// Ищем первый CSV файл
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

	return nil, errors.New("CSV файл не найден в TAR архиве")
}

// Read читает данные из текущего CSV файла.
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

// Close завершает чтение.
func (t *TarReader) Close() error {
	return nil // Нет дополнительных ресурсов для закрытия
}
