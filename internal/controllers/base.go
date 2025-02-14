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
	ProcessPrices(context.Context, io.Reader) (*models.ProcessResponse, error)
	GetAllProducts(context.Context) ([]models.Product, error)
}

// Log interface for logging
type Log interface {
	Info(string, ...zapcore.Field)
}

// BaseController struct for handling requests
type BaseController struct {
	ctx     context.Context
	storage Storage
	log     Log
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

	r.Group(func(r chi.Router) {
		r.Use(middleware.ArchiveTypeMiddleware)
		r.Post("/api/v0/prices", h.postPrices)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.CompressResponseMiddleware)
		r.Get("/api/v0/prices", h.getPrices)
	})

	return r
}

func (h *BaseController) postPrices(w http.ResponseWriter, r *http.Request) {
	response, err := h.storage.ProcessPrices(r.Context(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process prices: %v", err), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Return the result to the client
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *BaseController) getPrices(w http.ResponseWriter, r *http.Request) {
	products, err := h.storage.GetAllProducts(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve prices: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the Content-Type header
	w.Header().Set("Content-Type", "application/json")

	// Encode the data to JSON and send the response
	if err := json.NewEncoder(w).Encode(products); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
