package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/circleci/ex/config/secret"
	"github.com/circleci/ex/testing/dbfixture"
	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/imlogang/api-service/internal/db"
)

type fixture struct {
	api   *API
	store *db.Store
	db    *dbfixture.Fixture
}

// newFixture gives each test its own freshly created database with db.Schema
// applied, which is dropped when the test finishes.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := testcontext.Background()

	cfg := db.LoadConfig()
	fix := dbfixture.SetupDB(ctx, t, db.Schema, dbfixture.Connection{
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),
		User:     cfg.User,
		Password: secret.String(cfg.Password),
	})

	cfg.DB = fix.DBName
	store, err := db.New(ctx, cfg)
	assert.NilError(t, err)
	t.Cleanup(store.Close)

	a, err := New(ctx, store)
	assert.NilError(t, err)
	return &fixture{api: a, store: store, db: fix}
}

func (f *fixture) do(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var b []byte
	if body != nil {
		var err error
		b, err = json.Marshal(body)
		assert.NilError(t, err)
	}
	w := httptest.NewRecorder()
	f.api.Router.ServeHTTP(w, httptest.NewRequest(method, "http://localhost:8080"+target, bytes.NewReader(b)))
	return w
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

func TestAPI_CreateTable(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	for _, table := range []string{"beemoviebot", "random_table"} {
		t.Run(table, func(t *testing.T) {
			w := f.do(t, "POST", "/api/private/create_table", requestBody{TableName: table})
			assert.Equal(t, w.Code, 200)
			assert.Check(t, cmp.DeepEqual(decode(t, w), returnBody{TableCreated: table}))
		})
	}
}

func TestAPI_ListTables(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	w := f.do(t, "POST", "/api/private/create_table", requestBody{TableName: "beemoviebot"})
	assert.Equal(t, w.Code, 200)

	w = f.do(t, "GET", "/api/private/list_tables", nil)
	assert.Equal(t, w.Code, 200)
	tables := decode(t, w).Tables
	slices.Sort(tables)
	assert.Check(t, cmp.DeepEqual(tables, []string{"beemoviebot", "pokemon_scores"}))
}

func TestAPI_DeleteTable(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	w := f.do(t, "POST", "/api/private/create_table", requestBody{TableName: "random_table"})
	assert.Equal(t, w.Code, 200)

	w = f.do(t, "DELETE", "/api/private/delete_table", requestBody{TableName: "random_table"})
	assert.Equal(t, w.Code, 200)
	assert.Check(t, cmp.DeepEqual(decode(t, w), returnBody{TableDeleted: "random_table"}))

	w = f.do(t, "DELETE", "/api/private/delete_table", requestBody{TableName: "random_table"})
	assert.Equal(t, w.Code, 500)
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
	const evil = "injection_test; DROP TABLE pokemon_scores"

	w := f.do(t, "POST", "/api/private/create_table", requestBody{TableName: evil})
	assert.Equal(t, w.Code, 200)

	w = f.do(t, "GET", "/api/private/leaderboard?"+url.Values{"tablename": {"pokemon_scores; DROP TABLE pokemon_scores"}}.Encode(), nil)
	assert.Equal(t, w.Code, 500)

	w = f.do(t, "DELETE", "/api/private/delete_table", requestBody{TableName: evil})
	assert.Equal(t, w.Code, 200)

	w = f.do(t, "GET", "/api/private/list_tables", nil)
	assert.Check(t, cmp.DeepEqual(decode(t, w).Tables, []string{"pokemon_scores"}))
}

func TestAPI_ReadAnswerFromDBHandler(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	_, err := f.store.PutAnswerInDB("answer_test", "pikachu", "ANSWER", "POSITION", 2)
	assert.NilError(t, err)

	tests := []struct {
		name         string
		query        url.Values
		expectedCode int
		expectedResp string
	}{
		{
			name:         "Read answer column",
			query:        url.Values{"tablename": {"answer_test"}, "column": {"ANSWER"}},
			expectedCode: 200,
			expectedResp: "pikachu",
		},
		{
			name:         "Read integer column",
			query:        url.Values{"tablename": {"answer_test"}, "column": {"POSITION"}},
			expectedCode: 200,
			expectedResp: "2",
		},
		{
			name:         "Missing column",
			query:        url.Values{"tablename": {"answer_test"}},
			expectedCode: 400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := f.do(t, "GET", "/api/private/get_answer?"+tt.query.Encode(), nil)
			assert.Equal(t, w.Code, tt.expectedCode)
			if tt.expectedResp != "" {
				assert.Equal(t, w.Body.String(), tt.expectedResp)
			}
		})
	}
}
