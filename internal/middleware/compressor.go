package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/drstein77/priceanalyzer/internal/compress"
	"go.uber.org/zap"
)

// ArchiveTypeMiddleware — middleware для обработки ZIP и TAR архивов.
func ArchiveTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем параметр archiveType из query строк
		archiveType := r.URL.Query().Get("type")
		if archiveType != "tar" && archiveType != "zip" {
			archiveType = "zip"
		}

		// Применяем соответствующее сжатие
		compressMiddleware := CreateCompressMiddleware(archiveType)
		compressMiddleware(next).ServeHTTP(w, r)
	})
}

// CreateCompressMiddleware создаёт middleware для обработки архива заданного типа.
func CreateCompressMiddleware(archiveType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверяем Content-Type
			if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				http.Error(w, "Content-Type должен быть multipart/form-data", http.StatusBadRequest)
				return
			}

			// Парсим multipart форму с ограничением памяти 32MB
			err := r.ParseMultipartForm(32 << 20) // 32MB
			if err != nil {
				http.Error(w, "Не удалось разобрать multipart форму: "+err.Error(), http.StatusBadRequest)
				return
			}

			// Получаем файл из формы
			file, _, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Не удалось получить файл из формы: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer file.Close()

			var extractedData io.ReadCloser

			// В зависимости от типа архива используем соответствующий Reader
			switch archiveType {
			case "zip":
				extractedData, err = compress.NewZipReader(file)
			case "tar":
				extractedData, err = compress.NewTarReader(file)
			default:
				http.Error(w, "Неподдерживаемый тип архива", http.StatusBadRequest)
				return
			}

			if err != nil {
				http.Error(w, "Ошибка при обработке архива: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer extractedData.Close()

			// Заменяем r.Body на распакованные данные CSV
			r.Body = extractedData
			// Обновляем заголовок Content-Type на text/csv
			r.Header.Set("Content-Type", "text/csv")
			// Убираем Content-Length, так как он неизвестен после распаковки
			r.ContentLength = -1

			// Передаём управление следующему обработчику
			next.ServeHTTP(w, r)
		})
	}
}

// CompressResponseMiddleware создает middleware для упаковки ответов в ZIP архив.
func CompressResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем параметр archiveType из query строки
		archiveType := r.URL.Query().Get("archiveType")
		if archiveType != "zip" {
			// Если не указано, не упаковываем ответ
			next.ServeHTTP(w, r)
			return
		}

		// Создаем буфер для захвата ответа
		var buf bytes.Buffer
		// Используем ResponseWriter, который записывает в буфер
		rr := &responseRecorder{
			ResponseWriter: w,
			body:           &buf,
		}

		// Вызываем следующий обработчик с нашим ResponseRecorder
		next.ServeHTTP(rr, r)

		// Получаем статус код и заголовки
		statusCode := rr.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		// Упаковываем данные в ZIP архив
		var archiveBuffer bytes.Buffer
		zw, err := compress.NewZipWriter(&archiveBuffer, "data.csv")
		if err != nil {
			zap.L().Error("Failed to create ZIP writer", zap.Error(err))
			http.Error(w, "Ошибка при создании ZIP архива", http.StatusInternalServerError)
			return
		}

		// Пишем данные в архив
		_, err = zw.Write(buf.Bytes())
		if err != nil {
			zap.L().Error("Failed to write to ZIP archive", zap.Error(err))
			http.Error(w, "Ошибка при упаковке данных в ZIP архив", http.StatusInternalServerError)
			return
		}

		// Закрываем архив
		if err := zw.Close(); err != nil {
			zap.L().Error("Failed to close ZIP archive", zap.Error(err))
			http.Error(w, "Ошибка при завершении ZIP архива", http.StatusInternalServerError)
			return
		}

		// Устанавливаем заголовки
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="data.zip"`)
		w.WriteHeader(statusCode)

		// Отправляем архив клиенту
		_, err = w.Write(archiveBuffer.Bytes())
		if err != nil {
			zap.L().Error("Failed to write ZIP archive to response", zap.Error(err))
		}
	})
}

// responseRecorder захватывает ответ для последующей упаковки в архив.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// WriteHeader записывает статус код.
func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	// Не записываем сразу, чтобы дождаться упаковки данных
}

// Write записывает данные в буфер.
func (rr *responseRecorder) Write(b []byte) (int, error) {
	return rr.body.Write(b)
}
