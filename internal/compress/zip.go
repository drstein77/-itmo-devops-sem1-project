package compress

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"strings"
)

// ZipReader реализует io.ReadCloser для чтения содержимого CSV файла из ZIP архива.
type ZipReader struct {
	current io.ReadCloser
}

// NewZipReader создает новый ZipReader, извлекая первый найденный CSV файл из ZIP архива.
func NewZipReader(r io.ReadCloser) (*ZipReader, error) {
	defer r.Close()

	// Читаем весь архив в буфер
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}

	// Создаем zip.Reader
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return nil, err
	}

	// Ищем первый CSV файл
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

	return nil, errors.New("CSV файл не найден в ZIP архиве")
}

// Read читает данные из текущего CSV файла.
func (z *ZipReader) Read(p []byte) (int, error) {
	return z.current.Read(p)
}

// Close закрывает текущий CSV файл.
func (z *ZipReader) Close() error {
	return z.current.Close()
}

// ZipWriter реализует упаковку данных в ZIP архив.
type ZipWriter struct {
	zipWriter *zip.Writer
	file      io.Writer
}

// NewZipWriter создает новый ZipWriter с указанным именем файла внутри архива.
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

// Write записывает данные в файл внутри ZIP архива.
func (z *ZipWriter) Write(p []byte) (int, error) {
	return z.file.Write(p)
}

// Close закрывает ZIP архив.
func (z *ZipWriter) Close() error {
	return z.zipWriter.Close()
}
