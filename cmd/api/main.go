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
	"tenzoflow/internal/storage"
	"tenzoflow/internal/transport/rest"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Logger.Env, cfg.Logger.Level, os.Stdout)

	if err := run(cfg, log); err != nil {
		log.Error("application stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(cfg *config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("starting application",
		slog.String("env", cfg.Logger.Env),
		slog.String("addr", cfg.Server.Addr()),
		slog.Any("database", cfg.Database),
	)

	// Initial database and migrates UP
	db, err := storage.Open(ctx, cfg.Database, log)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("close database", slog.Any("error", err))
		}
	}()

	srv := &http.Server{
		Addr:              cfg.Server.Addr(),
		Handler:           rest.NewRouter(log),
		ReadHeaderTimeout: cfg.Server.Timeout,
		ReadTimeout:       cfg.Server.Timeout,
		WriteTimeout:      cfg.Server.Timeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}
	serverErr := make(chan error, 1)

	go func() {
		log.Info("http server is listening", slog.String("addr", srv.Addr))
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

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("server shutdown complete")
	return nil
}
