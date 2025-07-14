package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"gotest.tools/v3/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test HelloWorldHandler
func TestHelloWorldHandler(t *testing.T) {
	// Create a request to pass to the handler
	req, err := http.NewRequest("GET", "/api/private/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(HelloWorldHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("HelloWorldHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	expected := "Hello, World!"
	if rr.Body.String() != expected {
		t.Errorf("HelloWorldHandler returned wrong body: got %v want %v", rr.Body.String(), expected)
	}
}

// Test HealthCheckHandler
func TestHealthCheckHandler(t *testing.T) {
	// Create a request to pass to the handler
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Create an APIHandler instance with a context
	ctx := context.Background()
	apiHandler := NewAPIHandler(ctx) // or httpapi.NewAPIHandler(ctx) if in different package

	// Call the handler method
	apiHandler.HealthCheckHandler(rr, req)

	// Check if the status code is 200 OK
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("HealthCheckHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestAPI_CreateTable(t *testing.T) {
	tests := []struct {
		name    string
		request requestBody
	}{
		{
			name: "Beemoviebot Table",
			request: requestBody{
				TableName: "beemoviebot",
			},
		},
		{
			name: "Random Table",
			request: requestBody{
				TableName: "random_table",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)

			req := httptest.NewRequest("POST", "/api/private/create_table", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			CreateTableAPI(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			if _, exists := response["table_name_created"]; !exists {
				t.Errorf("Response missing 'table_name_created' key. Got: %v", response)
			}
		})
	}
}

func TestAPI_ListTables(t *testing.T) {
	tests := []struct {
		name  string
		table string
	}{
		{
			name:  "Beemoviebot Table",
			table: "beemoviebot",
		},
		{
			name:  "Random Table",
			table: "random_table",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/private/list_tables", nil)
			w := httptest.NewRecorder()
			ListTablesAPI(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var tables []string
			err := json.NewDecoder(w.Body).Decode(&tables)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			for _, tt := range tests {
				found := false
				for _, table := range tables {
					if table == tt.table {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Table '%s' not found in response", tt.table)
				}
			}
		})

	}
}
