package status

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

type DBChecker struct {
	db     *sql.DB
	driver string
	logger *slog.Logger
}

func NewDBChecker(db *sql.DB, driver string, logger *slog.Logger) *DBChecker {
	return &DBChecker{db: db, driver: driver, logger: logger}
}

func (c *DBChecker) Name() string {
	return "database"
}

func (c *DBChecker) Check(ctx context.Context) Component {
	start := time.Now()
	if err := c.db.PingContext(ctx); err != nil {
		c.logger.WarnContext(ctx, "database ping failed", slog.Any("error", err))
		return Component{
			Status:    StateDown,
			LatencyMs: millis(time.Since(start)),
			Error:     "database is unreachable",
			Details:   map[string]any{"driver": c.driver},
		}
	}
	latency := time.Since(start)
	stats := c.db.Stats()
	details := map[string]any{
		"driver":               c.driver,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"max_open_connections": stats.MaxOpenConnections,
	}

	if v, err := c.migrationVersion(ctx); err != nil {
		c.logger.WarnContext(ctx, "cannot read migration version", slog.Any("error", err))
	} else {
		details["migration_version"] = v
	}

	return Component{Status: StateOK, LatencyMs: millis(latency), Details: details}
}

func (c *DBChecker) migrationVersion(ctx context.Context) (int64, error) {
	var v int64
	err := c.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied`,
	).Scan(&v)
	return v, err
}

func millis(d time.Duration) float64 {
	return float64(d.Milliseconds()) / 1000
}
