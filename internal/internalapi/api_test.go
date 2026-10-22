package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/circleci/ex/testing/dbfixture"
	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/imlogang/api-service/internal/db"
	"github.com/imlogang/api-service/internal/db/dbtest"
)

type fixture struct {
	api   *API
	store *db.Store
	db    *dbfixture.Fixture
}

// newFixture gives each test its own freshly created and migrated database,
// which is dropped when the test finishes.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	store, fix := dbtest.NewMigratedStore(t)

	a, err := New(testcontext.Background(), store)
	assert.NilError(t, err)
	return &fixture{api: a, store: store, db: fix}
}

func (f *fixture) do(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return f.doAccept(t, method, target, "", body)
}

// doAccept is do with an Accept header; "" sends none.
func (f *fixture) doAccept(t *testing.T, method, target, accept string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var b []byte
	if body != nil {
		var err error
		b, err = json.Marshal(body)
		assert.NilError(t, err)
	}
	req := httptest.NewRequest(method, "http://localhost:8080"+target, bytes.NewReader(b))
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	w := httptest.NewRecorder()
	f.api.Router.ServeHTTP(w, req)
	return w
}

func decodeInto[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	assert.Check(t, cmp.Contains(w.Header().Get("Content-Type"), "application/json"))
	var resp T
	assert.NilError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp
}

func (f *fixture) addUser(t *testing.T, username string, score int) {
	t.Helper()
	_, err := f.db.AdminDB.ExecContext(testcontext.Background(),
		`INSERT INTO pokemon_scores (username, score) VALUES ($1, $2)`, username, score)
	assert.NilError(t, err)
}

func decode(t *testing.T, w *httptest.ResponseRecorder) returnBody {
	t.Helper()
	var resp returnBody
	assert.NilError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp
}

func scoreURL(table, username string) string {
	return "/api/private/get_current_score?" + url.Values{"tablename": {table}, "username": {username}}.Encode()
}

func TestAPI_HelloWorldHandler(t *testing.T) {
	t.Parallel()
	a, err := New(testcontext.Background(), nil)
	assert.NilError(t, err)

	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost:8080/api/private/hello", nil))
	assert.Equal(t, w.Code, 200)
	assert.Check(t, cmp.DeepEqual(decode(t, w), returnBody{Hello: "Hello world!"}))
}

func TestAPI_UpdateTableWithUser(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	// o'brien would have broken the old string-formatted INSERT.
	for _, user := range []string{"test-user", "test-user-2", "o'brien"} {
		t.Run(user, func(t *testing.T) {
			w := f.do(t, "PUT", "/api/private/update_table_with_user", requestBody{
				TableName: "pokemon_scores",
				User:      user,
			})
			assert.Equal(t, w.Code, 200)
			assert.Check(t, cmp.DeepEqual(decode(t, w), returnBody{AddedUser: user}))

			w = f.do(t, "GET", scoreURL("pokemon_scores", user), nil)
			assert.Equal(t, w.Body.String(), fmt.Sprintf("Score for %s: 0\n", user))
		})
	}
}

func TestAPI_GetCurrentScoreHandler(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "test-user", 3)

	t.Run("existing user", func(t *testing.T) {
		w := f.do(t, "GET", scoreURL("pokemon_scores", "test-user"), nil)
		assert.Equal(t, w.Code, 200)
		assert.Equal(t, w.Body.String(), "Score for test-user: 3\n")
	})

	t.Run("unknown user is created with a score of 0", func(t *testing.T) {
		w := f.do(t, "GET", scoreURL("pokemon_scores", "new-user"), nil)
		assert.Equal(t, w.Code, 200)
		assert.Equal(t, w.Body.String(), "Score for new-user: 0\n")

		var count int
		err := f.db.AdminDB.GetContext(testcontext.Background(), &count,
			`SELECT COUNT(*) FROM pokemon_scores WHERE username = 'new-user'`)
		assert.NilError(t, err)
		assert.Equal(t, count, 1)
	})

	t.Run("missing username", func(t *testing.T) {
		w := f.do(t, "GET", "/api/private/get_current_score?tablename=pokemon_scores", nil)
		assert.Equal(t, w.Code, 400)
	})

	t.Run("missing tablename", func(t *testing.T) {
		w := f.do(t, "GET", "/api/private/get_current_score?username=test-user", nil)
		assert.Equal(t, w.Code, 400)
	})
}

func TestAPI_UpdateScoreForUserHandler(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "test-user", 0)
	f.addUser(t, "test-user-2", 0)

	t.Run("updates only that user", func(t *testing.T) {
		w := f.do(t, "POST", "/api/private/update_user_score", requestBody{
			TableName: "pokemon_scores",
			User:      "test-user",
			Score:     5,
			Column:    "score",
		})
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decode(t, w), returnBody{
			UpdateAnswer: "the score for the user has been updated",
		}))

		w = f.do(t, "GET", scoreURL("pokemon_scores", "test-user"), nil)
		assert.Equal(t, w.Body.String(), "Score for test-user: 5\n")
		w = f.do(t, "GET", scoreURL("pokemon_scores", "test-user-2"), nil)
		assert.Equal(t, w.Body.String(), "Score for test-user-2: 0\n")
	})

	t.Run("username is not SQL", func(t *testing.T) {
		w := f.do(t, "POST", "/api/private/update_user_score", requestBody{
			TableName: "pokemon_scores",
			User:      "test-user' OR '1'='1",
			Score:     99,
			Column:    "score",
		})
		assert.Equal(t, w.Code, 200)

		w = f.do(t, "GET", scoreURL("pokemon_scores", "test-user-2"), nil)
		assert.Equal(t, w.Body.String(), "Score for test-user-2: 0\n")
	})

	for _, column := range []string{"id", "username", `score" = 0, "username`} {
		t.Run("rejects column "+column, func(t *testing.T) {
			w := f.do(t, "POST", "/api/private/update_user_score", requestBody{
				TableName: "pokemon_scores",
				User:      "test-user-2",
				Score:     7,
				Column:    column,
			})
			assert.Equal(t, w.Code, 400)
			assert.Check(t, cmp.Contains(decode(t, w).Error, "column cannot be updated"))
		})
	}
}

func TestAPI_LeaderboardHandler(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "bronze", 1)
	f.addUser(t, "gold", 9)
	f.addUser(t, "silver", 4)

	w := f.do(t, "GET", "/api/private/leaderboard?tablename=pokemon_scores", nil)
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(),
		"\nUsername: gold, Score: 9\nUsername: silver, Score: 4\nUsername: bronze, Score: 1\n")
}

func TestAPI_LeaderboardHandler_TopTenOnly(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for i := 1; i <= 12; i++ {
		f.addUser(t, fmt.Sprintf("user-%02d", i), i)
	}

	w := f.do(t, "GET", "/api/private/leaderboard?tablename=pokemon_scores", nil)
	assert.Equal(t, w.Code, 200)
	lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	assert.Equal(t, len(lines), 10)
	assert.Equal(t, lines[0], "Username: user-12, Score: 12")
	assert.Equal(t, lines[9], "Username: user-03, Score: 3")
}

func TestAPI_LeaderboardHandler_Empty(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	w := f.do(t, "GET", "/api/private/leaderboard?tablename=pokemon_scores", nil)
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "\nNo leaderboard data found.")

	w = f.do(t, "GET", "/api/private/leaderboard", nil)
	assert.Equal(t, w.Code, 400)
}

func TestAPI_TableNameIsNotSQL(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "test-user", 4)
	const evil = "pokemon_scores; DROP TABLE pokemon_scores"

	w := f.do(t, "GET", "/api/private/leaderboard?"+url.Values{"tablename": {evil}}.Encode(), nil)
	assert.Equal(t, w.Code, 500)

	w = f.do(t, "GET", scoreURL(evil, "test-user"), nil)
	assert.Equal(t, w.Code, 500)

	w = f.do(t, "GET", "/api/private/leaderboard?tablename=pokemon_scores", nil)
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "\nUsername: test-user, Score: 4\n")
}

func TestAPI_RemovedTableEndpoints(t *testing.T) {
	t.Parallel()
	a, err := New(testcontext.Background(), nil)
	assert.NilError(t, err)

	for _, route := range []string{
		"GET /api/private/list_tables",
		"POST /api/private/create_table",
		"DELETE /api/private/delete_table",
		"GET /api/private/get_answer",
	} {
		t.Run(route, func(t *testing.T) {
			method, path, _ := strings.Cut(route, " ")
			w := httptest.NewRecorder()
			a.Router.ServeHTTP(w, httptest.NewRequest(method, "http://localhost:8080"+path, nil))
			assert.Equal(t, w.Code, 404)
		})
	}
}

func TestAPI_GetCurrentScoreHandler_JSON(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "test-user", 3)

	t.Run("existing user", func(t *testing.T) {
		w := f.doAccept(t, "GET", scoreURL("pokemon_scores", "test-user"), "application/json", nil)
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decodeInto[scoreResponse](t, w), scoreResponse{Username: "test-user", Score: 3}))
	})

	t.Run("unknown user", func(t *testing.T) {
		w := f.doAccept(t, "GET", scoreURL("pokemon_scores", "new-user"), "application/json", nil)
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decodeInto[scoreResponse](t, w), scoreResponse{Username: "new-user", Score: 0}))
	})

	// node-fetch, which the bot uses, sends */* by default.
	for _, accept := range []string{"", "*/*", "text/plain"} {
		t.Run("text for Accept "+accept, func(t *testing.T) {
			w := f.doAccept(t, "GET", scoreURL("pokemon_scores", "test-user"), accept, nil)
			assert.Equal(t, w.Code, 200)
			assert.Equal(t, w.Body.String(), "Score for test-user: 3\n")
		})
	}
}

func TestAPI_LeaderboardHandler_JSON(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "bronze", 1)
	f.addUser(t, "gold", 9)
	f.addUser(t, "silver", 4)

	w := f.doAccept(t, "GET", "/api/private/leaderboard?tablename=pokemon_scores", "application/json", nil)
	assert.Equal(t, w.Code, 200)
	assert.Check(t, cmp.DeepEqual(decodeInto[leaderboardResponse](t, w), leaderboardResponse{
		Leaderboard: []db.LeaderboardEntry{
			{Username: "gold", Score: 9},
			{Username: "silver", Score: 4},
			{Username: "bronze", Score: 1},
		},
	}))

	w = f.doAccept(t, "GET", "/api/private/leaderboard?tablename=pokemon_scores", "*/*", nil)
	assert.Equal(t, w.Body.String(),
		"\nUsername: gold, Score: 9\nUsername: silver, Score: 4\nUsername: bronze, Score: 1\n")
}

func TestAPI_LeaderboardHandler_JSONEmpty(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	w := f.doAccept(t, "GET", "/api/private/leaderboard?tablename=pokemon_scores", "application/json", nil)
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, strings.TrimSpace(w.Body.String()), `{"leaderboard":[]}`)
}

func TestAPI_GetPokemonHandler(t *testing.T) {
	t.Parallel()
	a, err := New(testcontext.Background(), nil)
	assert.NilError(t, err)
	a.getPokemon = func(context.Context) (string, error) { return "meltan", nil }
	f := &fixture{api: a}

	t.Run("text keeps its trailing newline", func(t *testing.T) {
		w := f.do(t, "GET", "/api/private/get_pokemon", nil)
		assert.Equal(t, w.Code, 200)
		assert.Equal(t, w.Body.String(), "meltan\n")
	})

	t.Run("json", func(t *testing.T) {
		w := f.doAccept(t, "GET", "/api/private/get_pokemon", "application/json", nil)
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decodeInto[pokemonResponse](t, w), pokemonResponse{Pokemon: "meltan"}))
	})

	t.Run("pokeapi error", func(t *testing.T) {
		a.getPokemon = func(context.Context) (string, error) { return "", errors.New("pokeapi is down") }
		w := f.doAccept(t, "GET", "/api/private/get_pokemon", "application/json", nil)
		assert.Equal(t, w.Code, 500)
		assert.Check(t, cmp.Contains(decode(t, w).Error, "pokeapi is down"))
	})
}

func TestAPI_IncrementScoreHandler(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "test-user", 5)
	f.addUser(t, "bystander", 2)

	increment := func(t *testing.T, username string) scoreResponse {
		t.Helper()
		w := f.do(t, "POST", "/api/private/increment_score", requestBody{TableName: "pokemon_scores", User: username})
		assert.Equal(t, w.Code, 200)
		return decodeInto[scoreResponse](t, w)
	}

	t.Run("existing user", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(increment(t, "test-user"), scoreResponse{Username: "test-user", Score: 6}))
	})

	t.Run("new user starts at 1", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(increment(t, "new-user"), scoreResponse{Username: "new-user", Score: 1}))
		assert.Check(t, cmp.DeepEqual(increment(t, "new-user"), scoreResponse{Username: "new-user", Score: 2}))
	})

	t.Run("other users are untouched", func(t *testing.T) {
		w := f.do(t, "GET", scoreURL("pokemon_scores", "bystander"), nil)
		assert.Equal(t, w.Body.String(), "Score for bystander: 2\n")
	})

	t.Run("missing username", func(t *testing.T) {
		w := f.do(t, "POST", "/api/private/increment_score", requestBody{TableName: "pokemon_scores"})
		assert.Equal(t, w.Code, 400)
	})

	t.Run("unknown table", func(t *testing.T) {
		w := f.do(t, "POST", "/api/private/increment_score", requestBody{TableName: "nope", User: "test-user"})
		assert.Equal(t, w.Code, 500)
	})
}

// The bot used to read the score, add one, and write it back, so two correct
// answers at the same moment could lose a point. Increments must all land.
func TestAPI_IncrementScoreHandler_Concurrent(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	const n = 25
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := f.do(t, "POST", "/api/private/increment_score", requestBody{TableName: "pokemon_scores", User: "racer"})
			assert.Check(t, cmp.Equal(w.Code, 200))
		}()
	}
	wg.Wait()

	w := f.do(t, "GET", scoreURL("pokemon_scores", "racer"), nil)
	assert.Equal(t, w.Body.String(), fmt.Sprintf("Score for racer: %d\n", n))
}
