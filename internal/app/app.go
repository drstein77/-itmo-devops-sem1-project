package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/drstein77/priceanalyzer/internal/config"
	"github.com/drstein77/priceanalyzer/internal/logger"
)

type Server struct {
	srv *http.Server
	ctx context.Context
}

// NewServer creates a new Server instance with the provided context
func NewServer(ctx context.Context) *Server {
	server := new(Server)
	server.ctx = ctx
	return server
}

// Serve starts the server and handles signal interruption for graceful shutdown
func (server *Server) Serve() {
	// create and initialize a new option instance
	option := config.NewOptions()
	option.ParseFlags()

	// get a new logger
	nLogger, err := logger.NewLogger(option.LogLevel())
	if err != nil {
		log.Fatalln(err)
	}

	nLogger.Info("test")

	// // create router and mount routes
	// r := chi.NewRouter()
	// r.Use(reqLog.RequestLogger)
	// r.Mount("/", basecontr.Route())

	// // configure and start the server
	// server.srv = startServer(r, option.RunAddr())

	// Create a channel to receive interrupt signals (e.g., CTRL+C)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	// Block execution until a signal is received
	<-stopChan

}
