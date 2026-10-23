package db_test

import (
	"testing"

	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/imlogang/api-service/internal/db"
	"github.com/imlogang/api-service/internal/db/dbtest"
)

func TestStore_GetStanding(t *testing.T) {
	ctx := testcontext.Background()
	store, fix := dbtest.NewMigratedStore(t)
	_, err := fix.AdminDB.ExecContext(ctx,
		`INSERT INTO pokemon_scores (username, score) VALUES ('ash', 9), ('misty', 9), ('brock', 7)`)
	assert.NilError(t, err)

	st, err := store.GetStanding(ctx, "pokemon_scores", "brock")
	assert.NilError(t, err)
	assert.Check(t, cmp.DeepEqual(st, db.Standing{Rank: 3, Players: 3}))

	_, err = store.GetStanding(ctx, "pokemon_scores", "nobody")
	assert.Check(t, cmp.ErrorIs(err, db.ErrUserNotFound))

	_, err = store.GetStanding(ctx, "no_such_table", "brock")
	assert.Check(t, cmp.ErrorContains(err, "error finding the users rank"))
}
