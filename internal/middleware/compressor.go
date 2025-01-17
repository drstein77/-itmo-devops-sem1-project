package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/drstein77/priceanalyzer/internal/compress"
	"go.uber.org/zap"
)

// ArchiveTypeMiddleware is middleware for handling ZIP and TAR archives.
func ArchiveTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the archiveType parameter from the query string
		archiveType := r.URL.Query().Get("type")
		if archiveType != "tar" && archiveType != "zip" {
			archiveType = "zip"
		}

		// Apply the appropriate compression handling
		compressMiddleware := CreateCompressMiddleware(archiveType)
		compressMiddleware(next).ServeHTTP(w, r)
	})
}

// CreateCompressMiddleware creates middleware to handle archives of the specified type.
func CreateCompressMiddleware(archiveType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check Content-Type
			if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				http.Error(w, "Content-Type must be multipart/form-data", http.StatusBadRequest)
				return
			}

			// Parse the multipart form with a 32MB memory limit
			err := r.ParseMultipartForm(32 << 20) // 32MB
			if err != nil {
				http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
				return
			}

			// Retrieve the file from the form
			file, _, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Failed to retrieve file from form: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer file.Close()

			var extractedData io.ReadCloser

			// Use the appropriate reader based on the archive type
			switch archiveType {
			case "zip":
				extractedData, err = compress.NewZipReader(file)
			case "tar":
				extractedData, err = compress.NewTarReader(file)
			default:
				http.Error(w, "Unsupported archive type", http.StatusBadRequest)
				return
			}

			if err != nil {
				http.Error(w, "Error processing archive: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer extractedData.Close()

			// Replace r.Body with the unpacked CSV data
			r.Body = extractedData
			// Update the Content-Type header to text/csv
			r.Header.Set("Content-Type", "text/csv")
			// Remove Content-Length since it is unknown after unpacking
			r.ContentLength = -1

			// Pass control to the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// CompressResponseMiddleware creates middleware to compress responses into a ZIP archive.
func CompressResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a buffer to capture the response
		var buf bytes.Buffer
		// Use a ResponseWriter that writes to the buffer
		rr := &responseRecorder{
			ResponseWriter: w,
			body:           &buf,
		}

		// Call the next handler with the custom ResponseRecorder
		next.ServeHTTP(rr, r)

		// Get the status code and headers
		statusCode := rr.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		// Package the data into a ZIP archive
		var archiveBuffer bytes.Buffer
		zw, err := compress.NewZipWriter(&archiveBuffer, "data.csv")
		if err != nil {
			zap.L().Error("Failed to create ZIP writer", zap.Error(err))
			http.Error(w, "Error creating ZIP archive", http.StatusInternalServerError)
			return
		}

		// Write data into the archive
		_, err = zw.Write(buf.Bytes())
		if err != nil {
			zap.L().Error("Failed to write to ZIP archive", zap.Error(err))
			http.Error(w, "Error packing data into ZIP archive", http.StatusInternalServerError)
			return
		}

		// Close the archive
		if err := zw.Close(); err != nil {
			zap.L().Error("Failed to close ZIP archive", zap.Error(err))
			http.Error(w, "Error finalizing ZIP archive", http.StatusInternalServerError)
			return
		}

		// Set headers
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="data.zip"`)
		w.WriteHeader(statusCode)

		// Send the archive to the client
		_, err = w.Write(archiveBuffer.Bytes())
		if err != nil {
			zap.L().Error("Failed to write ZIP archive to response", zap.Error(err))
		}
	})
}

// responseRecorder captures the response for later packaging into an archive.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// WriteHeader records the status code.
func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	// Do not write immediately, wait until data is packed
}

// Write writes data to the buffer.
func (rr *responseRecorder) Write(b []byte) (int, error) {
	return rr.body.Write(b)
}
