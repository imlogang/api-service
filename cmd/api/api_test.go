package httpapi

import (
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

	// Call the handler
	handler := http.HandlerFunc(HealthCheckHandler)
	handler.ServeHTTP(rr, req)

	// Check if the status code is 200 OK
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("HealthCheckHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
