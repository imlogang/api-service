package db_test

import (
	"testing"

	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/imlogang/api-service/internal/db/dbtest"
)

func TestMigrate_FreshDatabase(t *testing.T) {
	t.Parallel()
	ctx := testcontext.Background()
	store, fix := dbtest.NewStore(t, "")

	applied, err := store.Migrate(ctx)
	assert.NilError(t, err)
	assert.Check(t, cmp.DeepEqual(applied, []string{"00001_create_pokemon_scores.sql"}))

	t.Run("running again applies nothing", func(t *testing.T) {
		applied, err := store.Migrate(ctx)
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(applied, 0))
	})

	t.Run("pokemon_scores works", func(t *testing.T) {
		_, err := fix.AdminDB.ExecContext(ctx, `INSERT INTO pokemon_scores (username) VALUES ('ash')`)
		assert.NilError(t, err)

		var score int
		err = fix.AdminDB.GetContext(ctx, &score, `SELECT score FROM pokemon_scores WHERE username = 'ash'`)
		assert.NilError(t, err)
		assert.Equal(t, score, 0)
	})
}

// Production's pokemon_scores was created before migrations existed, either by
// the old startup code or by the old create_table + add-columns path. The
// first migration has to leave both shapes, and their data, alone.

func TestMigrate_ExistingTableFromStartupCode(t *testing.T) {
	t.Parallel()
	testMigrateKeepsExistingTable(t, `
		CREATE TABLE pokemon_scores (
			id SERIAL PRIMARY KEY,
			username TEXT UNIQUE,
			score INTEGER NOT NULL DEFAULT 0
		);`)
}

func TestMigrate_ExistingTableFromCreateTableEndpoint(t *testing.T) {
	t.Parallel()
	testMigrateKeepsExistingTable(t, `
		CREATE TABLE pokemon_scores (id SERIAL PRIMARY KEY, name TEXT);
		ALTER TABLE pokemon_scores ADD COLUMN "username" VARCHAR(255);
		ALTER TABLE pokemon_scores ADD COLUMN "score" INTEGER;`)
}

func testMigrateKeepsExistingTable(t *testing.T, schema string) {
	t.Helper()
	ctx := testcontext.Background()
	store, fix := dbtest.NewStore(t, schema+`
		INSERT INTO pokemon_scores (username, score) VALUES ('misty', 12), ('brock', 7);`)

	applied, err := store.Migrate(ctx)
	assert.NilError(t, err)
	assert.Check(t, cmp.Len(applied, 1))

	var total int
	err = fix.AdminDB.GetContext(ctx, &total, `SELECT SUM(score) FROM pokemon_scores`)
	assert.NilError(t, err)
	assert.Equal(t, total, 19)
}
