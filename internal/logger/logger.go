package logger

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

const (
	EnvLocal = "local"
	EnvProd  = "prod"
)

func SetupLogger(env string, output io.Writer) *slog.Logger {
	if output == nil {
		output = os.Stdout
	}

	var handler slog.Handler
	switch env {
	case EnvLocal:
		handler = tint.NewTextHandler(output, &tint.Options{Level: slog.LevelDebug, TimeFormat: time.TimeOnly, NoColor: false})
	case EnvProd:
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo, AddSource: true})
	default:
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
