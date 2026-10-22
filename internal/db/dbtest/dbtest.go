// Package dbtest creates throwaway databases for tests.
package dbtest

import (
	"net"
	"testing"

	"github.com/circleci/ex/config/secret"
	"github.com/circleci/ex/testing/dbfixture"
	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"

	"github.com/imlogang/api-service/internal/db"
)

// NewStore creates a fresh database with the given schema applied (use "" for
// an empty one) and returns a Store connected to it. The database is dropped
// when the test finishes. Migrations are not run.
func NewStore(t *testing.T, schema string) (*db.Store, *dbfixture.Fixture) {
	t.Helper()
	ctx := testcontext.Background()

	cfg := db.LoadConfig()
	fix := dbfixture.SetupDB(ctx, t, schema, dbfixture.Connection{
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),
		User:     cfg.User,
		Password: secret.String(cfg.Password),
	})

	cfg.DB = fix.DBName
	store, err := db.New(ctx, cfg)
	assert.NilError(t, err)
	t.Cleanup(store.Close)
	return store, fix
}

// NewMigratedStore is NewStore on an empty database with all migrations applied.
func NewMigratedStore(t *testing.T) (*db.Store, *dbfixture.Fixture) {
	t.Helper()
	store, fix := NewStore(t, "")
	_, err := store.Migrate(testcontext.Background())
	assert.NilError(t, err)
	return store, fix
}
