package httpapi

import (
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
			u, err := url.Parse("http://localhost:8082/api/private/hello")
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

//func TestAPI_CreateTable(t *testing.T) {
//	tests := []struct {
//		name    string
//		request requestBody
//	}{
//		{
//			name: "Beemoviebot Table",
//			request: requestBody{
//				TableName: "beemoviebot",
//			},
//		},
//		{
//			name: "Random Table",
//			request: requestBody{
//				TableName: "random_table",
//			},
//		},
//	}
//	for _, tt := range tests {
//		tt := tt
//		t.Run(tt.name, func(t *testing.T) {
//			body, _ := json.Marshal(tt.request)
//
//			req := httptest.NewRequest("POST", "/api/private/create_table", bytes.NewReader(body))
//			req.Header.Set("Content-Type", "application/json")
//			w := httptest.NewRecorder()
//			CreateTableAPI(w, req)
//			assert.Equal(t, http.StatusOK, w.Code)
//
//			var response map[string]string
//			err := json.NewDecoder(w.Body).Decode(&response)
//			if err != nil {
//				t.Fatalf("Failed to decode response: %v", err)
//			}
//			if _, exists := response["table_name_created"]; !exists {
//				t.Errorf("Response missing 'table_name_created' key. Got: %v", response)
//			}
//		})
//	}
//}

//func TestAPI_ListTables(t *testing.T) {
//	tests := []struct {
//		name  string
//		table string
//	}{
//		{
//			name:  "Beemoviebot Table",
//			table: "beemoviebot",
//		},
//		{
//			name:  "Random Table",
//			table: "random_table",
//		},
//	}
//	for _, tt := range tests {
//		tt := tt
//		t.Run(tt.name, func(t *testing.T) {
//			req := httptest.NewRequest("GET", "/api/private/list_tables", nil)
//			w := httptest.NewRecorder()
//			ListTablesAPI(w, req)
//
//			assert.Equal(t, http.StatusOK, w.Code)
//			var tables []string
//			err := json.NewDecoder(w.Body).Decode(&tables)
//			if err != nil {
//				t.Fatalf("Failed to decode response: %v", err)
//			}
//			for _, tt := range tests {
//				found := false
//				for _, table := range tables {
//					if table == tt.table {
//						found = true
//						break
//					}
//				}
//				if !found {
//					t.Errorf("Table '%s' not found in response", tt.table)
//				}
//			}
//		})
//
//	}
//}
