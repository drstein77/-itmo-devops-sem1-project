package middleware

import (
	"io"
	"net/http"
	"strings"

	"github.com/drstein77/priceanalyzer/internal/compress"
)

func ArchiveTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the archiveType query parameter
		archiveType := r.URL.Query().Get("archiveType")
		if archiveType != "tar" && archiveType != "zip" {
			archiveType = "zip" // Default value
		}

		// Dynamically apply compression middleware
		compressMiddleware := CreateCompressMiddleware(archiveType)
		compressMiddleware(next).ServeHTTP(w, r)
	})
}

func CreateCompressMiddleware(compressionType string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// By default set the original http.ResponseWriter
			ow := w

			// Check if the client can accept compressed data
			acceptEncoding := r.Header.Get("Accept-Encoding")
			supportsCompression := strings.Contains(acceptEncoding, compressionType)
			if supportsCompression {
				var cw io.Closer
				if compressionType == "tar" {
					cw = compress.NewTarWriter(w)
				} else if compressionType == "zip" {
					cw = compress.NewZipWriter(w)
				} else {
					h.ServeHTTP(w, r)
					return
				}
				ow = cw.(http.ResponseWriter)
				defer cw.Close()
			}

			// Check if the client sent compressed data
			contentEncoding := r.Header.Get("Content-Encoding")
			if contentEncoding == compressionType {
				var cr io.ReadCloser
				var err error
				if compressionType == "tar" {
					cr, err = compress.NewTarReader(r.Body)
				} else if compressionType == "zip" {
					cr, err = compress.NewZipReader(r.Body)
				} else {
					h.ServeHTTP(w, r)
					return
				}

				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				r.Body = cr
				defer cr.Close()
			}

			// Transfer control to the handler
			h.ServeHTTP(ow, r)
		})
	}
}
