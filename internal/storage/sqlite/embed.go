package sqlite

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

func migrationsFS() (fs.FS, error) {
	sub, err := fs.Sub(embeddedMigrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("sqlite migrations fs: %w", err)
	}
	return sub, nil
}
