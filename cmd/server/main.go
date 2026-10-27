package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"multithreadredis/internal/config"
	"multithreadredis/internal/observability"
	"multithreadredis/internal/pool"
	"multithreadredis/internal/server"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	// `server healthcheck [flags]` is used by the Docker HEALTHCHECK.
	healthcheck := len(args) > 0 && args[0] == "healthcheck"
	if healthcheck {
		args = args[1:]
	}

	cfg, err := config.Load(args)
	if err != nil {
		return err
	}

	if healthcheck {
		return observability.Healthcheck(cfg.MetricsAddr)
	}

	if err := observability.SetupLogger(cfg.LogLevel); err != nil {
		return err
	}

	// SIGTERM is what `docker stop` sends.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	workerPool := pool.New(cfg.GoroutinePoolSize, cfg.MaxWaitingConnections, cfg.BufferSize)

	srv, err := server.Listen(cfg.BindedPort, workerPool)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.BindedPort, err)
	}

	metrics := observability.NewHTTPServer(cfg.MetricsAddr, workerPool)

	go func() {
		if err := metrics.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Metrics server failed", "error", err)
			stop()
		}
	}()

	slog.Info("server started",
		"addr", srv.Addr().String(),
		"metrics_addr", cfg.MetricsAddr,
		"pool_size", cfg.GoroutinePoolSize,
		"max_waiting_conns", cfg.MaxWaitingConnections,
	)

	err = srv.Serve(ctx)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	metrics.Shutdown(shutdownCtx)

	slog.Info("Server stopped")

	return err
}
