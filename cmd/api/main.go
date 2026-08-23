package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pplmx/aurora/internal/api"
	"github.com/pplmx/aurora/internal/config"
	"github.com/pplmx/aurora/internal/logger"
)

const shutdownTimeout = 15 * time.Second

// HTTP server timeouts (v1.60). net/http defaults leave ReadHeaderTimeout,
// ReadTimeout, WriteTimeout and IdleTimeout at zero, meaning a connection that
// trickles headers or a body (slowloris / slow-body DoS) can hold a handler
// (and its goroutine) open indefinitely, and idle keep-alive connections are
// never reaped. Keeping them bounded bounds the resources any single client
// can consume; most API operations are fast, so even conservative values are
// not a throughput constraint. ReadHeaderTimeout also independently guards
// against slow-header attacks even before a body is read.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

// newServer creates the HTTP server for the API. Extracted as a pure
// constructor so the timeout configuration is unit-testable (ISS-065,
// TASK-073); it does not Listen, so a test can assert the configured
// timeouts without binding a socket.
func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init()

	srv, err := api.NewServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}
	// Release the SQLite/event-store handles NewServer opened when the server
	// exits (graceful or forced shutdown), matching srv.Close() semantics.
	defer func() { _ = srv.Close() }()

	// Start the oracle fetch scheduler so enabled sources are refreshed on
	// their configured interval (v1.15 Oracle Scheduled Fetching). Cancelled
	// at shutdown.
	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()
	stopScheduler := srv.StartOracleScheduler(schedCtx, config.OracleSchedulerCheckInterval())
	if stopScheduler != nil {
		defer stopScheduler()
	}

	router := srv.Router()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := newServer(addr, router)

	go func() {
		logger.Info().Str("addr", addr).Msg("Starting API server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info().Str("signal", sig.String()).Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server shutdown error")
		if err := server.Close(); err != nil {
			logger.Error().Err(err).Msg("Server force close error")
		}
	}

	logger.Info().Msg("Server stopped")
}
