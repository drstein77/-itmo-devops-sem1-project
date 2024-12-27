package controllers

import (
	"context"

	"github.com/go-chi/chi"
	"go.uber.org/zap/zapcore"
)

// Storage interface for database operations
type Storage interface {
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

	// r.Post("/api/message", h.testPost)
	// r.Get("/api/messages", h.testGet)
	return r
}
