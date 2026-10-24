package db_test

import (
	"testing"

	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/imlogang/api-service/internal/db"
	"github.com/imlogang/api-service/internal/db/dbtest"
)

func TestStore_Collection(t *testing.T) {
	ctx := testcontext.Background()
	store, _ := dbtest.NewMigratedStore(t)

	add := func(game, username string, item int, shiny bool) bool {
		t.Helper()
		added, err := store.AddToCollection(ctx, game, username, item, shiny)
		assert.NilError(t, err)
		return added
	}

	assert.Check(t, add("pokemon", "ash", 25, false), "first pikachu is new")
	assert.Check(t, !add("pokemon", "ash", 25, false), "second pikachu isn't")
	assert.Check(t, add("pokemon", "ash", 1, true))
	assert.Check(t, !add("pokemon", "ash", 1, false), "a normal catch keeps the shiny")
	assert.Check(t, !add("pokemon", "ash", 25, true), "a shiny catch of a caught item isn't new")
	assert.Check(t, add("pokemon", "misty", 25, false), "collections are per user")
	assert.Check(t, add("trivia", "ash", 25, false), "and per game")

	got, err := store.GetCollection(ctx, "pokemon", "ash")
	assert.NilError(t, err)
	assert.Check(t, cmp.DeepEqual(got, db.Collection{Caught: 2, Shiny: 2}))

	got, err = store.GetCollection(ctx, "pokemon", "brock")
	assert.NilError(t, err)
	assert.Check(t, cmp.DeepEqual(got, db.Collection{}))

	_, err = store.AddToCollection(ctx, "", "ash", 1, false)
	assert.Check(t, cmp.ErrorContains(err, "must not be empty"))
	_, err = store.GetCollection(ctx, "pokemon", "")
	assert.Check(t, cmp.ErrorContains(err, "must not be empty"))
}
