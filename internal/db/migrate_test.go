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
	assert.Check(t, cmp.DeepEqual(applied, []string{
		"00001_create_pokemon_scores.sql",
		"00002_unique_pokemon_scores_username.sql",
	}))

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
// migrations have to leave both shapes, and their data, alone, and leave
// username unique so increment_score's ON CONFLICT works.

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
	assert.Check(t, cmp.Len(applied, 2))

	var total int
	err = fix.AdminDB.GetContext(ctx, &total, `SELECT SUM(score) FROM pokemon_scores`)
	assert.NilError(t, err)
	assert.Equal(t, total, 19)

	_, err = fix.AdminDB.ExecContext(ctx, `INSERT INTO pokemon_scores (username, score) VALUES ('misty', 1)`)
	assert.Check(t, cmp.ErrorContains(err, "pokemon_scores_username_key"))
}

// If a table without a unique index already has duplicate usernames, 00002
// can't add one. Migrate fails (so startup fails and the old pod keeps
// serving) and the existing rows are untouched.
func TestMigrate_DuplicateUsernamesBlockUniqueIndex(t *testing.T) {
	t.Parallel()
	ctx := testcontext.Background()
	store, fix := dbtest.NewStore(t, `
		CREATE TABLE pokemon_scores (id SERIAL PRIMARY KEY, name TEXT);
		ALTER TABLE pokemon_scores ADD COLUMN "username" VARCHAR(255);
		ALTER TABLE pokemon_scores ADD COLUMN "score" INTEGER;
		INSERT INTO pokemon_scores (username, score) VALUES ('misty', 12), ('misty', 3);`)

	_, err := store.Migrate(ctx)
	assert.Check(t, cmp.ErrorContains(err, `could not create unique index "pokemon_scores_username_key"`))

	var rows int
	err = fix.AdminDB.GetContext(ctx, &rows, `SELECT COUNT(*) FROM pokemon_scores`)
	assert.NilError(t, err)
	assert.Equal(t, rows, 2)
}
