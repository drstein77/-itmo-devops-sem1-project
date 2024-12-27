package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/drstein77/priceanalyzer/internal/app"
)

func main() {
	// Create a root context with cancellation capability
	ctx, cancel := context.WithCancel(context.Background())

	// Create a channel to handle signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Start the server
	server := app.NewServer(ctx)
	go func() {
		// Wait for a signal
		sig := <-signalCh
		log.Printf("Received signal: %+v", sig)

		// Shutdown the server
		// server.Shutdown()

		// Cancel the context
		cancel()
	}()

	// Start the server
	server.Serve()
}
