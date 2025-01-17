package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/drstein77/priceanalyzer/internal/app"
)

func main() {
	const shutdownTimeout = 5 * time.Second
	// Create a root context with the possibility of cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create a channel for signal handling
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Start the server
	server := app.NewServer(ctx)
	go func() {
		// Wait for a signal
		sig := <-signalCh
		server.Log.Info(fmt.Sprintf("Received signal: %+v", sig))

		// Perform graceful server shutdown
		server.Shutdown(shutdownTimeout)

		// Cancel the context
		cancel()
	}()

	// Start the server
	server.Serve()
}
