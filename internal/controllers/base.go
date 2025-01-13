package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/drstein77/priceanalyzer/internal/middleware"
	"github.com/drstein77/priceanalyzer/internal/models"
	"github.com/go-chi/chi"
	"go.uber.org/zap/zapcore"
)

// Storage interface for database operations
type Storage interface {
	ProcessPrices(io.Reader) (*models.ProcessResponse, error)
}

// Log interface for logging
type Log interface {
	Info(string, ...zapcore.Field)
}

// BaseController struct for handling requests
type BaseController struct {
	ctx            context.Context
	storage        Storage
	defaultEndTime func() string
	log            Log
}

// NewBaseController creates a new BaseController instance
func NewBaseController(ctx context.Context, storage Storage, log Log) *BaseController {
	instance := &BaseController{
		ctx:     ctx,
		storage: storage,
		log:     log,
	}

	return instance
}

// Route sets up the routes for the BaseController
func (h *BaseController) Route() *chi.Mux {
	r := chi.NewRouter()

	// Применяем ArchiveTypeMiddleware только для POST запросов
	r.With(middleware.ArchiveTypeMiddleware).Post("/api/v0/prices", h.postPrices)

	// Применяем CompressMiddleware для GET запросов
	r.With(middleware.CreateCompressMiddleware("zip")).Get("/api/v0/prices", h.getPrices)

	return r
}

func (h *BaseController) postPrices(w http.ResponseWriter, r *http.Request) {
	// Передаём тело запроса (stream CSV) в storage
	response, err := h.storage.ProcessPrices(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process prices: %v", err), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Возвращаем результат клиенту
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *BaseController) getPrices(w http.ResponseWriter, r *http.Request) {
	// ...
}
