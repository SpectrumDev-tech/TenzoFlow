package main

import (
	"TenzoFlow/internal/config"
	"TenzoFlow/internal/logger"
	"log/slog"
	"os"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Logger.Env, os.Stdout)

	log.Info("starting server",
		slog.Int("Port", cfg.Server.Port),
		slog.String("Host", cfg.Server.Host),
		slog.String("Driver", cfg.Database.Driver),
	)
}
