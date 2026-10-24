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
	"github.com/imlogang/api-service/internal/games"
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

func (f *fixture) score(t *testing.T, username string) int {
	t.Helper()
	w := f.do(t, "GET", scoreURL("pokemon_scores", username), nil)
	assert.Equal(t, w.Code, 200)
	return decodeInto[scoreResponse](t, w).Score
}

func (f *fixture) leaderboard(t *testing.T) []db.LeaderboardEntry {
	t.Helper()
	w := f.do(t, "GET", "/api/private/leaderboard?tablename=pokemon_scores", nil)
	assert.Equal(t, w.Code, 200)
	return decodeInto[leaderboardResponse](t, w).Leaderboard
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

// Users are created on first use, by get_current_score or increment_score.
func TestAPI_GetCurrentScoreHandler_CreatesUsers(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	// o'brien would have broken the old string-formatted INSERT.
	for _, user := range []string{"test-user", "o'brien"} {
		t.Run(user, func(t *testing.T) {
			assert.Equal(t, f.score(t, user), 0)
			assert.Equal(t, f.score(t, user), 0)

			var count int
			err := f.db.AdminDB.GetContext(testcontext.Background(), &count,
				`SELECT COUNT(*) FROM pokemon_scores WHERE username = $1`, user)
			assert.NilError(t, err)
			assert.Equal(t, count, 1)
		})
	}
}

// Several commands can look up a brand-new user at once; each should create
// the user or find it, never fail on the other's insert.
func TestAPI_GetCurrentScoreHandler_ConcurrentNewUser(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	const n = 25
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := f.do(t, "GET", scoreURL("pokemon_scores", "newcomer"), nil)
			assert.Check(t, cmp.Equal(w.Code, 200), w.Body.String())
		}()
	}
	wg.Wait()

	var count int
	err := f.db.AdminDB.GetContext(testcontext.Background(), &count,
		`SELECT COUNT(*) FROM pokemon_scores WHERE username = 'newcomer'`)
	assert.NilError(t, err)
	assert.Equal(t, count, 1)
}

// Equal scores share a rank and the next rank skips ahead; players on 0
// count too.
func TestAPI_GetCurrentScoreHandler_Rank(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "ash", 9)
	f.addUser(t, "misty", 9)
	f.addUser(t, "brock", 7)
	f.addUser(t, "gary", 0)

	for user, want := range map[string]scoreResponse{
		"ash":   {Username: "ash", Score: 9, Rank: 1, Players: 4},
		"misty": {Username: "misty", Score: 9, Rank: 1, Players: 4},
		"brock": {Username: "brock", Score: 7, Rank: 3, Players: 4},
		"gary":  {Username: "gary", Score: 0, Rank: 4, Players: 4},
	} {
		w := f.do(t, "GET", scoreURL("pokemon_scores", user), nil)
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decodeInto[scoreResponse](t, w), want))
	}
}

func TestAPI_GetCurrentScoreHandler(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "test-user", 3)

	t.Run("existing user", func(t *testing.T) {
		w := f.do(t, "GET", scoreURL("pokemon_scores", "test-user"), nil)
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decodeInto[scoreResponse](t, w), scoreResponse{Username: "test-user", Score: 3, Rank: 1, Players: 1}))
	})

	// node-fetch, which the bot uses, sends */* unless told otherwise.
	for _, accept := range []string{"*/*", "text/plain", "application/json"} {
		t.Run("json for Accept "+accept, func(t *testing.T) {
			w := f.doAccept(t, "GET", scoreURL("pokemon_scores", "test-user"), accept, nil)
			assert.Equal(t, w.Code, 200)
			assert.Check(t, cmp.DeepEqual(decodeInto[scoreResponse](t, w), scoreResponse{Username: "test-user", Score: 3, Rank: 1, Players: 1}))
		})
	}

	t.Run("unknown user is created with a score of 0", func(t *testing.T) {
		w := f.do(t, "GET", scoreURL("pokemon_scores", "new-user"), nil)
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decodeInto[scoreResponse](t, w), scoreResponse{Username: "new-user", Score: 0, Rank: 2, Players: 2}))

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

	update := func(t *testing.T, body requestBody) *httptest.ResponseRecorder {
		t.Helper()
		return f.do(t, "POST", "/api/private/update_user_score", body)
	}

	t.Run("updates only that user", func(t *testing.T) {
		w := update(t, requestBody{TableName: "pokemon_scores", User: "test-user", Score: new(5), Column: "score"})
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decode(t, w), returnBody{
			UpdateAnswer: "the score for the user has been updated",
		}))

		assert.Equal(t, f.score(t, "test-user"), 5)
		assert.Equal(t, f.score(t, "test-user-2"), 0)
	})

	t.Run("resets a score to 0", func(t *testing.T) {
		f.addUser(t, "reset-me", 12)
		w := update(t, requestBody{TableName: "pokemon_scores", User: "reset-me", Score: new(0), Column: "score"})
		assert.Equal(t, w.Code, 200)
		assert.Equal(t, f.score(t, "reset-me"), 0)
	})

	t.Run("unknown user", func(t *testing.T) {
		w := update(t, requestBody{TableName: "pokemon_scores", User: "nobody", Score: new(3), Column: "score"})
		assert.Equal(t, w.Code, 404)
		assert.Check(t, cmp.Contains(decode(t, w).Error, "user not found"))
	})

	t.Run("username is not SQL", func(t *testing.T) {
		w := update(t, requestBody{TableName: "pokemon_scores", User: "test-user' OR '1'='1", Score: new(99), Column: "score"})
		assert.Equal(t, w.Code, 404)

		assert.Equal(t, f.score(t, "test-user-2"), 0)
	})

	t.Run("missing score is not a reset", func(t *testing.T) {
		w := update(t, requestBody{TableName: "pokemon_scores", User: "test-user", Column: "score"})
		assert.Equal(t, w.Code, 400)
		assert.Check(t, cmp.Equal(decode(t, w).Error, "score required"))
		assert.Equal(t, f.score(t, "test-user"), 5)
	})

	t.Run("negative score", func(t *testing.T) {
		w := update(t, requestBody{TableName: "pokemon_scores", User: "test-user", Score: new(-1), Column: "score"})
		assert.Equal(t, w.Code, 400)
		assert.Check(t, cmp.Contains(decode(t, w).Error, "must not be negative"))
		assert.Equal(t, f.score(t, "test-user"), 5)
	})

	for _, column := range []string{"id", "username", `score" = 0, "username`} {
		t.Run("rejects column "+column, func(t *testing.T) {
			w := update(t, requestBody{TableName: "pokemon_scores", User: "test-user-2", Score: new(7), Column: column})
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

	assert.Check(t, cmp.DeepEqual(f.leaderboard(t), []db.LeaderboardEntry{
		{Username: "gold", Score: 9},
		{Username: "silver", Score: 4},
		{Username: "bronze", Score: 1},
	}))
}

func TestAPI_LeaderboardHandler_TopTenOnly(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for i := 1; i <= 12; i++ {
		f.addUser(t, fmt.Sprintf("user-%02d", i), i)
	}

	leaderboard := f.leaderboard(t)
	assert.Equal(t, len(leaderboard), 10)
	assert.Equal(t, leaderboard[0], db.LeaderboardEntry{Username: "user-12", Score: 12})
	assert.Equal(t, leaderboard[9], db.LeaderboardEntry{Username: "user-03", Score: 3})
}

// Equal scores are listed by username, which also decides who makes the top
// ten when there's a tie at the cutoff.
func TestAPI_LeaderboardHandler_TiesByUsername(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addUser(t, "leader", 20)
	for _, name := range []string{"kirby", "alpha", "zed", "mario", "bravo", "luigi", "yoshi", "peach", "toad", "wario"} {
		f.addUser(t, name, 5)
	}

	var names []string
	for _, e := range f.leaderboard(t) {
		names = append(names, e.Username)
	}
	assert.Check(t, cmp.DeepEqual(names, []string{
		"leader", "alpha", "bravo", "kirby", "luigi", "mario", "peach", "toad", "wario", "yoshi",
	}))
}

func TestAPI_LeaderboardHandler_Empty(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	// An empty list, not null, so the bot can tell "nobody yet" from a bad response.
	w := f.do(t, "GET", "/api/private/leaderboard?tablename=pokemon_scores", nil)
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, strings.TrimSpace(w.Body.String()), `{"leaderboard":[]}`)

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

	assert.Check(t, cmp.DeepEqual(f.leaderboard(t), []db.LeaderboardEntry{{Username: "test-user", Score: 4}}))
}

func TestAPI_RemovedEndpoints(t *testing.T) {
	t.Parallel()
	a, err := New(testcontext.Background(), nil)
	assert.NilError(t, err)

	for _, route := range []string{
		"GET /api/private/list_tables",
		"POST /api/private/create_table",
		"DELETE /api/private/delete_table",
		"GET /api/private/get_answer",
		"GET /api/private/get_pokemon",
		"PUT /api/private/update_table_with_user",
	} {
		t.Run(route, func(t *testing.T) {
			method, path, _ := strings.Cut(route, " ")
			w := httptest.NewRecorder()
			a.Router.ServeHTTP(w, httptest.NewRequest(method, "http://localhost:8080"+path, nil))
			assert.Equal(t, w.Code, 404)
		})
	}
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
		assert.Check(t, cmp.DeepEqual(increment(t, "test-user"), scoreResponse{Username: "test-user", Score: 6, Rank: 1, Players: 2}))
	})

	t.Run("new user starts at 1", func(t *testing.T) {
		// Scores are now 6, 2 and 1, then 6, 2 and 2, a tie with bystander.
		assert.Check(t, cmp.DeepEqual(increment(t, "new-user"), scoreResponse{Username: "new-user", Score: 1, Rank: 3, Players: 3}))
		assert.Check(t, cmp.DeepEqual(increment(t, "new-user"), scoreResponse{Username: "new-user", Score: 2, Rank: 2, Players: 3}))
	})

	t.Run("other users are untouched", func(t *testing.T) {
		assert.Equal(t, f.score(t, "bystander"), 2)
	})

	t.Run("points", func(t *testing.T) {
		w := f.do(t, "POST", "/api/private/increment_score",
			requestBody{TableName: "pokemon_scores", User: "bystander", Points: new(3)})
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.Equal(decodeInto[scoreResponse](t, w).Score, 5))
		// And for a new user.
		w = f.do(t, "POST", "/api/private/increment_score",
			requestBody{TableName: "pokemon_scores", User: "shiny-hunter", Points: new(3)})
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.Equal(decodeInto[scoreResponse](t, w).Score, 3))
	})

	t.Run("points below 1", func(t *testing.T) {
		for _, p := range []int{0, -2} {
			w := f.do(t, "POST", "/api/private/increment_score",
				requestBody{TableName: "pokemon_scores", User: "test-user", Points: new(p)})
			assert.Equal(t, w.Code, 400)
			assert.Check(t, cmp.Contains(decode(t, w).Error, "points must be at least 1"))
		}
		assert.Equal(t, f.score(t, "test-user"), 6)
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

	assert.Equal(t, f.score(t, "racer"), n)
}

type stubGame struct {
	round games.Round
	size  int
	err   error
}

func (s stubGame) Round(context.Context) (games.Round, error) { return s.round, s.err }
func (s stubGame) Size(context.Context) (int, error)          { return s.size, s.err }

func TestAPI_Collection(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.api.games = map[string]games.Game{
		"pokemon": stubGame{size: 1025},
		"broken":  stubGame{err: errors.New("pokeapi is down")},
	}

	collect := func(t *testing.T, username string, id int, shiny bool) collectionResponse {
		t.Helper()
		w := f.do(t, "POST", "/api/private/add_to_collection",
			collectRequest{Game: "pokemon", Username: username, ID: id, Shiny: shiny})
		assert.Equal(t, w.Code, 200)
		return decodeInto[collectionResponse](t, w)
	}
	get := func(t *testing.T, query string) *httptest.ResponseRecorder {
		t.Helper()
		return f.do(t, "GET", "/api/private/get_collection?"+query, nil)
	}

	t.Run("empty", func(t *testing.T) {
		w := get(t, "game=pokemon&username=ash")
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decodeInto[collectionResponse](t, w),
			collectionResponse{Username: "ash", Game: "pokemon", Total: 1025}))
	})

	t.Run("add", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(collect(t, "ash", 25, false),
			collectionResponse{Username: "ash", Game: "pokemon", New: true, Caught: 1, Total: 1025}))
		assert.Check(t, cmp.DeepEqual(collect(t, "ash", 25, false),
			collectionResponse{Username: "ash", Game: "pokemon", Caught: 1, Total: 1025}))
		assert.Check(t, cmp.DeepEqual(collect(t, "ash", 1, true),
			collectionResponse{Username: "ash", Game: "pokemon", New: true, Caught: 2, ShinyCaught: 1, Total: 1025}))
	})

	t.Run("get", func(t *testing.T) {
		w := get(t, "game=pokemon&username=ash")
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decodeInto[collectionResponse](t, w),
			collectionResponse{Username: "ash", Game: "pokemon", Caught: 2, ShinyCaught: 1, Total: 1025}))
	})

	t.Run("bad requests", func(t *testing.T) {
		for name, req := range map[string]collectRequest{
			"no game":      {Username: "ash", ID: 1},
			"no username":  {Game: "pokemon", ID: 1},
			"no id":        {Game: "pokemon", Username: "ash"},
			"unknown game": {Game: "tetris", Username: "ash", ID: 1},
		} {
			t.Run(name, func(t *testing.T) {
				w := f.do(t, "POST", "/api/private/add_to_collection", req)
				assert.Equal(t, w.Code, 400)
			})
		}
		for _, q := range []string{"username=ash", "game=pokemon", "game=tetris&username=ash"} {
			t.Run(q, func(t *testing.T) {
				assert.Equal(t, get(t, q).Code, 400)
			})
		}
	})

	t.Run("game error", func(t *testing.T) {
		w := get(t, "game=broken&username=ash")
		assert.Equal(t, w.Code, 500)
		assert.Check(t, cmp.Contains(decode(t, w).Error, "pokeapi is down"))
	})
}

func TestAPI_GetRoundHandler(t *testing.T) {
	t.Parallel()
	a, err := New(testcontext.Background(), nil)
	assert.NilError(t, err)
	round := games.Round{
		Answer:  "meltan",
		ID:      808,
		Options: []string{"pikachu", "meltan", "eevee", "mew"},
		Image:   "https://example.com/808.png",
		Points:  1,
	}
	a.games = map[string]games.Game{
		"pokemon": stubGame{round: round},
		"broken":  stubGame{err: errors.New("pokeapi is down")},
	}
	f := &fixture{api: a}

	t.Run("ok", func(t *testing.T) {
		w := f.do(t, "GET", "/api/private/get_round?game=pokemon", nil)
		assert.Equal(t, w.Code, 200)
		assert.Check(t, cmp.DeepEqual(decodeInto[roundResponse](t, w), roundResponse{Game: "pokemon", Round: round}))
	})

	t.Run("fields are flat", func(t *testing.T) {
		w := f.do(t, "GET", "/api/private/get_round?game=pokemon", nil)
		assert.Equal(t, strings.TrimSpace(w.Body.String()),
			`{"game":"pokemon","answer":"meltan","id":808,"options":["pikachu","meltan","eevee","mew"],"image":"https://example.com/808.png","shiny":false,"points":1}`)
	})

	t.Run("missing game", func(t *testing.T) {
		w := f.do(t, "GET", "/api/private/get_round", nil)
		assert.Equal(t, w.Code, 400)
		assert.Check(t, cmp.Equal(decode(t, w).Error, "game required"))
	})

	t.Run("unknown game", func(t *testing.T) {
		w := f.do(t, "GET", "/api/private/get_round?game=tetris", nil)
		assert.Equal(t, w.Code, 400)
		assert.Check(t, cmp.Equal(decode(t, w).Error, `unknown game "tetris"`))
	})

	t.Run("game error", func(t *testing.T) {
		w := f.do(t, "GET", "/api/private/get_round?game=broken", nil)
		assert.Equal(t, w.Code, 500)
		assert.Check(t, cmp.Contains(decode(t, w).Error, "pokeapi is down"))
	})
}

func TestAPI_MalformedBodies(t *testing.T) {
	t.Parallel()
	a, err := New(testcontext.Background(), nil)
	assert.NilError(t, err)

	for _, route := range []string{
		"POST /api/private/increment_score",
		"POST /api/private/update_user_score",
		"POST /api/private/add_to_collection",
	} {
		for name, body := range map[string]string{
			"empty":      "",
			"not json":   "username=test-user",
			"wrong type": `{"table_name": "pokemon_scores", "username": 7}`,
			"truncated":  `{"table_name": "pokemon_scores", "username": "test-`,
		} {
			t.Run(route+" "+name, func(t *testing.T) {
				method, path, _ := strings.Cut(route, " ")
				w := httptest.NewRecorder()
				a.Router.ServeHTTP(w, httptest.NewRequest(method, "http://localhost:8080"+path, strings.NewReader(body)))
				assert.Equal(t, w.Code, 400)
			})
		}
	}
}
