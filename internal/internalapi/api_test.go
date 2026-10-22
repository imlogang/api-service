package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/circleci/ex/testing/testcontext"
	"github.com/imlogang/api-service/internal/db"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"testing"
)

var testStore *db.Store

func TestMain(m *testing.M) {
	store, err := db.New(testcontext.Background(), db.LoadConfig())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	testStore = store
	code := m.Run()
	store.Close()
	os.Exit(code)
}

func TestAPI_HelloWorldHandler(t *testing.T) {
	ctx := testcontext.Background()
	tests := []struct {
		name         string
		expectedResp returnBody
	}{
		{
			name: "Hello world Handler",
			expectedResp: returnBody{
				Hello: "Hello world!",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx, testStore)
			assert.NilError(t, err)
			w := httptest.NewRecorder()
			u, err := url.Parse("http://localhost:8080/api/private/hello")
			assert.NilError(t, err)

			req := httptest.NewRequest("GET", u.String(), nil)
			a.Router.ServeHTTP(w, req)

			var resp returnBody
			err = json.NewDecoder(w.Body).Decode(&resp)
			assert.NilError(t, err)
			assert.Check(t, cmp.DeepEqual(resp, tt.expectedResp))
		})
	}
}

func TestAPI_CreateTable(t *testing.T) {
	ctx := testcontext.Background()
	tests := []struct {
		name         string
		request      requestBody
		expectedResp returnBody
	}{
		{
			name:         "Beemoviebot Table",
			request:      requestBody{TableName: "beemoviebot"},
			expectedResp: returnBody{TableCreated: "beemoviebot"},
		},
		{
			name:         "Random Table",
			request:      requestBody{TableName: "random_table"},
			expectedResp: returnBody{TableCreated: "random_table"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx, testStore)
			assert.NilError(t, err)
			w := httptest.NewRecorder()
			u, err := url.Parse("http://localhost:8080/api/private/create_table")
			assert.NilError(t, err)
			body, err := json.Marshal(tt.request)
			assert.NilError(t, err)

			req := httptest.NewRequest("POST", u.String(), bytes.NewReader(body))
			a.Router.ServeHTTP(w, req)

			var resp returnBody
			err = json.NewDecoder(w.Body).Decode(&resp)
			assert.NilError(t, err)

			assert.Check(t, cmp.DeepEqual(resp, tt.expectedResp))
		})
	}
}

func TestAPI_ListTables(t *testing.T) {
	ctx := testcontext.Background()
	tests := []struct {
		name           string
		expectedTables returnBody
	}{
		{
			name:           "Return All Tables",
			expectedTables: returnBody{Tables: []string{"pokemon_scores", "beemoviebot", "random_table"}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx, testStore)
			assert.NilError(t, err)
			w := httptest.NewRecorder()
			u, err := url.Parse("http://localhost:8080/api/private/list_tables")
			assert.NilError(t, err)

			req := httptest.NewRequest("GET", u.String(), nil)
			var resp returnBody

			a.Router.ServeHTTP(w, req)

			err = json.NewDecoder(w.Body).Decode(&resp)
			assert.NilError(t, err)
			assert.Check(t, cmp.DeepEqual(resp, tt.expectedTables))
		})

	}
}

func TestAPI_UpdateTableWithUser(t *testing.T) {
	ctx := testcontext.Background()
	tests := []struct {
		name         string
		request      requestBody
		expectedResp returnBody
	}{
		{
			name: "Update Table With test-user",
			request: requestBody{
				TableName: "pokemon_scores",
				User:      "test-user",
			},
			expectedResp: returnBody{AddedUser: "test-user"},
		},
		{
			name: "Update Table With test-user-2",
			request: requestBody{
				TableName: "pokemon_scores",
				User:      "test-user-2",
			},
			expectedResp: returnBody{AddedUser: "test-user-2"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx, testStore)
			assert.NilError(t, err)
			w := httptest.NewRecorder()
			u, err := url.Parse("http://localhost:8080/api/private/update_table_with_user")
			assert.NilError(t, err)
			request, err := json.Marshal(tt.request)
			assert.NilError(t, err)
			req := httptest.NewRequest("PUT", u.String(), bytes.NewReader(request))
			a.Router.ServeHTTP(w, req)
			var resp returnBody
			err = json.NewDecoder(w.Body).Decode(&resp)
			assert.NilError(t, err)
			assert.Check(t, cmp.DeepEqual(resp, tt.expectedResp))
		})
	}
}

func TestAPI_GetCurrentScoreHandler(t *testing.T) {
	ctx := testcontext.Background()
	tests := []struct {
		name         string
		expectedResp string
		username     string
		score        int
		tableName    string
	}{
		{
			name:         "Get current score for test-user",
			username:     "test-user",
			score:        0,
			tableName:    "pokemon_scores",
			expectedResp: "Score for test-user: 0\n",
		},
		{
			name:         "Get current score for test-user-2",
			username:     "test-user-2",
			score:        0,
			tableName:    "pokemon_scores",
			expectedResp: "Score for test-user-2: 0\n",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx, testStore)
			assert.NilError(t, err)
			w := httptest.NewRecorder()
			formatedURL := fmt.Sprintf("http://localhost:8080/api/private/get_current_score?username=%s&tablename=%s", tt.username, tt.tableName)
			u, err := url.Parse(formatedURL)
			assert.NilError(t, err)
			req := httptest.NewRequest("GET", u.String(), nil)
			a.Router.ServeHTTP(w, req)
			assert.Check(t, cmp.DeepEqual(w.Body.String(), tt.expectedResp))

		})
	}
}

func TestAPI_UpdateScoreForUserHandler(t *testing.T) {
	ctx := testcontext.Background()
	tests := []struct {
		name         string
		request      requestBody
		expectedResp returnBody
	}{
		{
			name: "Update Table for test-user",
			request: requestBody{
				TableName: "pokemon_scores",
				User:      "test-user",
				Score:     1,
				Column:    "score",
			},
			expectedResp: returnBody{
				UpdateAnswer: "the score for the user has been updated",
			},
		},
		{
			name: "Update Table for test-user-2",
			request: requestBody{
				TableName: "pokemon_scores",
				User:      "test-user-2",
				Score:     1,
				Column:    "score",
			},
			expectedResp: returnBody{
				UpdateAnswer: "the score for the user has been updated",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx, testStore)
			assert.NilError(t, err)
			w := httptest.NewRecorder()
			u, err := url.Parse("http://localhost:8080/api/private/update_user_score")
			assert.NilError(t, err)
			request, err := json.Marshal(tt.request)
			assert.NilError(t, err)
			req := httptest.NewRequest("POST", u.String(), bytes.NewReader(request))
			a.Router.ServeHTTP(w, req)
			var resp returnBody
			err = json.NewDecoder(w.Body).Decode(&resp)
			assert.NilError(t, err)
			assert.Check(t, cmp.DeepEqual(resp, tt.expectedResp))
		})
	}
}

func TestAPI_LeaderboardHandler(t *testing.T) {
	ctx := testcontext.Background()
	tests := []struct {
		name         string
		expectedResp string
		tableName    string
		leaderboard  string
	}{
		{
			name:         "Get leaderboard",
			tableName:    "pokemon_scores",
			expectedResp: "\nUsername: test-user, Score: 1\nUsername: test-user-2, Score: 1\n",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx, testStore)
			assert.NilError(t, err)
			w := httptest.NewRecorder()
			formatedURL := fmt.Sprintf("http://localhost:8080/api/private/leaderboard?tablename=%s", tt.tableName)
			u, err := url.Parse(formatedURL)
			assert.NilError(t, err)
			req := httptest.NewRequest("GET", u.String(), nil)
			a.Router.ServeHTTP(w, req)
			assert.Check(t, cmp.DeepEqual(w.Body.String(), tt.expectedResp))
		})
	}
}

// The tests below share database state with the tests above and run after
// them, so the rows they add don't change the leaderboard expectations.

func doRequest(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	a, err := New(testcontext.Background(), testStore)
	assert.NilError(t, err)

	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		assert.NilError(t, err)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}

	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, httptest.NewRequest(method, "http://localhost:8080"+target, reader))
	return w
}

func scoreURL(table, username string) string {
	return "/api/private/get_current_score?" + url.Values{"tablename": {table}, "username": {username}}.Encode()
}

func TestAPI_UpdateScoreForUserHandler_UsernameIsNotSQL(t *testing.T) {
	w := doRequest(t, "POST", "/api/private/update_user_score", requestBody{
		TableName: "pokemon_scores",
		User:      "test-user' OR '1'='1",
		Score:     99,
		Column:    "score",
	})
	assert.Equal(t, w.Code, 200)

	w = doRequest(t, "GET", scoreURL("pokemon_scores", "test-user"), nil)
	assert.Equal(t, w.Body.String(), "Score for test-user: 1\n")
	w = doRequest(t, "GET", scoreURL("pokemon_scores", "test-user-2"), nil)
	assert.Equal(t, w.Body.String(), "Score for test-user-2: 1\n")
}

func TestAPI_UpdateScoreForUserHandler_RejectsOtherColumns(t *testing.T) {
	for _, column := range []string{"id", "username", `score" = 0, "username`} {
		t.Run(column, func(t *testing.T) {
			w := doRequest(t, "POST", "/api/private/update_user_score", requestBody{
				TableName: "pokemon_scores",
				User:      "test-user",
				Score:     5,
				Column:    column,
			})
			assert.Equal(t, w.Code, 400)
			var resp returnBody
			assert.NilError(t, json.NewDecoder(w.Body).Decode(&resp))
			assert.Check(t, cmp.Contains(resp.Error, "column cannot be updated"))
		})
	}

	w := doRequest(t, "GET", scoreURL("pokemon_scores", "test-user"), nil)
	assert.Equal(t, w.Body.String(), "Score for test-user: 1\n")
}

func TestAPI_UpdateTableWithUser_QuoteInUsername(t *testing.T) {
	w := doRequest(t, "PUT", "/api/private/update_table_with_user", requestBody{
		TableName: "pokemon_scores",
		User:      "o'brien",
	})
	assert.Equal(t, w.Code, 200)
	var resp returnBody
	assert.NilError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Check(t, cmp.DeepEqual(resp, returnBody{AddedUser: "o'brien"}))

	w = doRequest(t, "GET", scoreURL("pokemon_scores", "o'brien"), nil)
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "Score for o'brien: 0\n")
}

func TestAPI_TableNameIsNotSQL(t *testing.T) {
	const evil = "injection_test; DROP TABLE pokemon_scores"

	w := doRequest(t, "POST", "/api/private/create_table", requestBody{TableName: evil})
	assert.Equal(t, w.Code, 200)

	w = doRequest(t, "GET", "/api/private/leaderboard?"+url.Values{"tablename": {"pokemon_scores; DROP TABLE pokemon_scores"}}.Encode(), nil)
	assert.Equal(t, w.Code, 500)

	w = doRequest(t, "DELETE", "/api/private/delete_table", requestBody{TableName: evil})
	assert.Equal(t, w.Code, 200)

	w = doRequest(t, "GET", "/api/private/list_tables", nil)
	var resp returnBody
	assert.NilError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Check(t, cmp.Contains(resp.Tables, "pokemon_scores"))
	assert.Check(t, !slices.Contains(resp.Tables, evil))
}

func TestAPI_DeleteTable(t *testing.T) {
	w := doRequest(t, "DELETE", "/api/private/delete_table", requestBody{TableName: "random_table"})
	assert.Equal(t, w.Code, 200)
	var resp returnBody
	assert.NilError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Check(t, cmp.DeepEqual(resp, returnBody{TableDeleted: "random_table"}))

	w = doRequest(t, "DELETE", "/api/private/delete_table", requestBody{TableName: "random_table"})
	assert.Equal(t, w.Code, 500)
}

func TestAPI_ReadAnswerFromDBHandler(t *testing.T) {
	_, err := testStore.PutAnswerInDB("answer_test", "pikachu", "ANSWER", "POSITION", 2)
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
			w := doRequest(t, "GET", "/api/private/get_answer?"+tt.query.Encode(), nil)
			assert.Equal(t, w.Code, tt.expectedCode)
			if tt.expectedResp != "" {
				assert.Equal(t, w.Body.String(), tt.expectedResp)
			}
		})
	}
}

func TestAPI_MissingQueryParams(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "Score without username", target: "/api/private/get_current_score?tablename=pokemon_scores"},
		{name: "Score without tablename", target: "/api/private/get_current_score?username=test-user"},
		{name: "Leaderboard without tablename", target: "/api/private/leaderboard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(t, "GET", tt.target, nil)
			assert.Equal(t, w.Code, 400)
		})
	}
}
