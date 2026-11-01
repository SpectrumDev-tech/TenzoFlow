package logger

import (
	"context"
	"fmt"
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

type ctxKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

func ParseLevel(s string) (slog.Level, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return 0, fmt.Errorf("invalid log level %q (want debug, info, warn, or error): %w", s, err)
	}
	return l, nil
}

func levelFor(env, override string) slog.Level {
	if override != "" {
		if l, err := ParseLevel(override); err == nil {
			return l
		}
	}
	if env == EnvLocal {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func New(env, level string, output io.Writer) *slog.Logger {
	if output == nil {
		output = os.Stdout
	}
	lvl := levelFor(env, level)
	var handler slog.Handler
	switch env {
	case EnvLocal:
		handler = tint.NewTextHandler(output, &tint.Options{Level: lvl, TimeFormat: time.TimeOnly, NoColor: false})
	case EnvProd:
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{Level: lvl, AddSource: true})
	default:
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{Level: lvl})
	}
	return slog.New(contextHandler{handler})
}

type contextHandler struct {
	slog.Handler
}

func SetupLogger(env, level string, output io.Writer) *slog.Logger {
	l := New(env, level, output)
	slog.SetDefault(l)
	return l
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
