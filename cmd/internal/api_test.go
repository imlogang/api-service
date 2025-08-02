package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"net/http/httptest"
	"net/url"
	"testing"
)

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
			a, err := New(ctx)
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
		{
			name:         "Pokemon Scores Table",
			request:      requestBody{TableName: "pokemon_scores"},
			expectedResp: returnBody{TableCreated: "pokemon_scores"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Run(tt.name, func(t *testing.T) {
				a, err := New(ctx)
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
			expectedTables: returnBody{Tables: []string{"beemoviebot", "random_table", "pokemon_scores"}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx)
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx)
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
			name:         "Get current score",
			username:     "test-user",
			score:        0,
			tableName:    "pokemon_scores",
			expectedResp: "Score for test-user: 0\n",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx)
			assert.NilError(t, err)
			w := httptest.NewRecorder()
			u, err := url.Parse("http://localhost:8080/api/private/get_current_score?username=test-user&tablename=pokemon_scores")
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
				Column:    "SCORE",
			},
			expectedResp: returnBody{
				UpdateAnswer: "the score for the user has been updated",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(ctx)
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
