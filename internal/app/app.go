package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/drstein77/priceanalyzer/internal/config"
	"github.com/drstein77/priceanalyzer/internal/controllers"
	"github.com/drstein77/priceanalyzer/internal/dbkeeper"
	"github.com/drstein77/priceanalyzer/internal/logger"
	"github.com/drstein77/priceanalyzer/internal/middleware"
	"github.com/drstein77/priceanalyzer/internal/storage"
	"github.com/go-chi/chi"
)

type Server struct {
	srv *http.Server
	ctx context.Context
	Log *logger.Logger
}

// NewServer creates a new Server instance with the provided context
func NewServer(ctx context.Context) *Server {
	return &Server{ctx: ctx}
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
	server.Log = nLogger

	// initialize the keeper instance
	keeper := initializeKeeper(server.ctx, option.DataBaseDSN, nLogger)
	if keeper == nil {
		nLogger.Debug("Failed to initialize keeper")
	}
	defer keeper.Close()

	// initialize the storage instance
	memoryStorage := initializeStorage(server.ctx, keeper, nLogger)
	if memoryStorage == nil {
		nLogger.Debug("Failed to initialize storage")
	}

	// create a new controller to process incoming requests
	basecontr := initializeBaseController(server.ctx, memoryStorage, nLogger)

	// get a middleware for logging requests
	reqLog := middleware.NewReqLog(nLogger)

	// create router and mount routes
	r := chi.NewRouter()
	r.Use(reqLog.RequestLogger)
	r.Mount("/", basecontr.Route())

	// configure and start the server
	server.srv = startServer(r, option.RunAddr())

	select {
	case <-server.ctx.Done():
		return
	}
}

// initializeKeeper initializes a DBKeeper instance
func initializeKeeper(ctx context.Context, dataBaseDSN func() string, logger *logger.Logger) *dbkeeper.DBKeeper {
	return dbkeeper.NewDBKeeper(ctx, dataBaseDSN, logger)
}

// initializeStorage initializes a MemoryStorage instance
func initializeStorage(ctx context.Context, keeper storage.Keeper, logger *logger.Logger) *storage.MemoryStorage {
	return storage.NewMemoryStorage(ctx, keeper, logger)
}

// initializeBaseController initializes a BaseController instance
func initializeBaseController(ctx context.Context, storage *storage.MemoryStorage,
	logger *logger.Logger,
) *controllers.BaseController {
	return controllers.NewBaseController(ctx, storage, logger)
}

// startServer configures and starts an HTTP server with the provided router and address
func startServer(router chi.Router, address string) *http.Server {
	const (
		oneMegabyte = 1 << 20
		readTimeout = 3 * time.Second
	)

	server := &http.Server{
		Addr:                         address,
		Handler:                      router,
		ReadHeaderTimeout:            readTimeout,
		WriteTimeout:                 readTimeout,
		IdleTimeout:                  readTimeout,
		ReadTimeout:                  readTimeout,
		MaxHeaderBytes:               oneMegabyte, // 1 MB
		DisableGeneralOptionsHandler: false,
		TLSConfig:                    nil,
		TLSNextProto:                 nil,
		ConnState:                    nil,
		ErrorLog:                     nil,
		BaseContext:                  nil,
		ConnContext:                  nil,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln(err)
		}
	}()

	return server
}

// Shutdown gracefully shuts down the server
func (server *Server) Shutdown(timeout time.Duration) {
	ctxShutDown, cancel := context.WithTimeout(server.ctx, timeout)
	defer cancel()

	server.Log.Info("attempting to stop the server")

	if server.srv != nil {
		if err := server.srv.Shutdown(ctxShutDown); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Printf("server Shutdown Failed: %s", err)
				return
			}
		}
		server.Log.Info("server stopped")
	}
	server.Log.Info("server exited properly")
}
