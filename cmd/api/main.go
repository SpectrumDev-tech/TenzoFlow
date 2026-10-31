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
	"tenzoflow/internal/config"
	"tenzoflow/internal/logger"
	"tenzoflow/internal/storage/sqlite"
	"tenzoflow/internal/transport/rest"
	"time"
)

const (
	idleTimeout     = 60 * time.Second
	shutdownTimeout = 30 * time.Second
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Logger.Env, os.Stdout)

	if err := run(cfg, log); err != nil {
		log.Error("application stopped with error", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("starting server",
		slog.Int("Port", cfg.Server.Port),
		slog.String("Host", cfg.Server.Host),
		slog.String("Driver", cfg.Database.Driver),
	)
}

func run(cfg *config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initial database and migrates UP
	db, err := sqlite.Open(ctx, cfg.Database.Host)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	log.Info(
		"database is ready",
		slog.String("driver", cfg.Database.Driver),
		slog.String("host", cfg.Server.Host),
	)

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           rest.NewRouter(log),
		ReadHeaderTimeout: cfg.Server.Timeout,
		ReadTimeout:       cfg.Server.Timeout,
		WriteTimeout:      cfg.Server.Timeout,
		IdleTimeout:       idleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("starting server", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server failed: %w", err)
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("server shutdown complete")
	return nil
}
