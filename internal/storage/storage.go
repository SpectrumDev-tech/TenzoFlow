package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"tenzoflow/internal/config"
	"tenzoflow/internal/storage/postgres"
	"tenzoflow/internal/storage/sqlite"
)

var ErrSupportedDriver = errors.New("unsupported database driver")

func Open(ctx context.Context, cfg config.DatabaseCfg, log *slog.Logger) (*sql.DB, error) {
	var (
		db              *sql.DB
		err             error
		applyMigrations func(context.Context, *sql.DB, *slog.Logger) error
	)
	switch cfg.Driver {
	case config.DriverSQLite:
		db, err = sqlite.Open(ctx, cfg.SQLite.Path)
		applyMigrations = sqlite.Migrate
	case config.DriverPostgres:
		pg := cfg.Postgres
		db, err = postgres.Open(ctx, pg.DSN(), postgres.PoolOptions{
			MaxOpenConns:    pg.MaxOpenConns,
			MaxIdleConns:    pg.MaxIdleConns,
			ConnMaxLifetime: pg.ConnMaxLifetime,
		})
		applyMigrations = postgres.Migrate
	default:
		return nil, fmt.Errorf("%w: %q", ErrSupportedDriver, cfg.Driver)
	}
	if err != nil {
		return nil, err
	}
	if err := applyMigrations(ctx, db, log); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
