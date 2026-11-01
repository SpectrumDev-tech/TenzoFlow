package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"
)

func Up(ctx context.Context, db *sql.DB, dialect goose.Dialect, fsys fs.FS, log *slog.Logger) error {
	provider, err := goose.NewProvider(dialect, db, fsys)
	if err != nil {
		return fmt.Errorf("create migrate provider: %w", err)
	}

	result, err := provider.Up(ctx)
	for _, r := range result {
		log.InfoContext(ctx, "migrations applied",
			slog.String("file", r.Source.Path),
			slog.Duration("duration", r.Duration),
		)
	}
	if err != nil {
		return fmt.Errorf("applied migrations: %w", err)
	}
	if len(result) == 0 {
		log.InfoContext(ctx, "database schema is up to date")
	}
	return nil
}
