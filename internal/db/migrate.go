package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Schema changes live in migrations/ as numbered goose files. To add a table,
// add a new file (e.g. 00002_create_trivia_scores.sql); never edit one that
// has already been deployed.
//
//go:embed migrations/*.sql
var migrations embed.FS

// Migrate applies any migrations that haven't run yet and returns their names.
// A Postgres advisory lock stops two pods from migrating at the same time.
func (s *Store) Migrate(ctx context.Context) ([]string, error) {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, err
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("error creating migration lock: %w", err)
	}

	// goose needs a database/sql handle; closing it leaves the pool open.
	sqlDB := stdlib.OpenDBFromPool(s.pool)
	defer func() { _ = sqlDB.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fsys, goose.WithSessionLocker(locker))
	if err != nil {
		return nil, fmt.Errorf("error loading migrations: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("error applying migrations: %w", err)
	}

	applied := make([]string, 0, len(results))
	for _, r := range results {
		applied = append(applied, r.Source.Path)
	}
	return applied, nil
}
