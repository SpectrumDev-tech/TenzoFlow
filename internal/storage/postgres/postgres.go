package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"tenzoflow/internal/storage/migrate"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/pressly/goose/v3"
)

const pingTimeout = 5 * time.Second

type PoolOptions struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func Open(ctx context.Context, dsn string, pool PoolOptions) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}
	if pool.MaxOpenConns > 0 {
		db.SetMaxOpenConns(pool.MaxOpenConns)
	}
	if pool.MaxIdleConns > 0 {
		db.SetMaxIdleConns(pool.MaxIdleConns)
	}
	if pool.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(pool.ConnMaxLifetime)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping postgres (is server up? `try task db:up`): %w)", err)
	}
	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	fsys, err := migrationsFS()
	if err != nil {
		return err
	}
	return migrate.Up(ctx, db, goose.DialectPostgres, fsys, log)
}
