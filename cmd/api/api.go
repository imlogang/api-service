package httpapi

import (
	"encoding/json"
	"go-api/cmd/db"
	"go-api/cmd/deluge"
	"io"
	"log"
	"net/http"
	"fmt"
)

type TorrentRequest struct {
	Parameters struct {
		URL string `json:"url"`
	} `json:"parameters"`
}

// addTorrentHandler handles the HTTP request to add a torrent file
func AddTorrentHandler(w http.ResponseWriter, r *http.Request) {
	// Read the request body and check for errors
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close() // Close the body after reading to avoid resource leak

	// Log the raw request body for debugging purposes
	log.Printf("Received request body: %s", string(bodyBytes))

	// Initialize the struct to hold the parsed request
	var req TorrentRequest

	// Decode the JSON request body into the struct using the read bytes
	err = json.Unmarshal(bodyBytes, &req)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the URL is empty and return an error
	if req.Parameters.URL == "" {
		http.Error(w, "Torrent URL is required", http.StatusBadRequest)
		return
	}

	// Log the parsed torrent URL for debugging purposes
	log.Printf("Parsed torrent URL: %s", req.Parameters.URL)

	// Call the AddTorrentFile function from the deluge package with the parsed URL
	result, err := deluge.AuthAndDownloadTorrent(req.Parameters.URL)
	if err != nil {
		http.Error(w, "Error downloading torrent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the result as JSON and check for any error during encoding
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}


func HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	// Write the response and check for any error, but discard 'n' as it's not needed
	_, err := w.Write([]byte("Hello, World!"))
	if err != nil {
		// If there is an error, return an internal server error response
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

//Health check for k8s
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func GetRoot(w http.ResponseWriter, r *http.Request) {
	// Write the string and check for any error
	_, err := io.WriteString(w, "This is my website!\n")
	if err != nil {
		// If there is an error, return an internal server error response
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func ListTablesAPI(w http.ResponseWriter, r *http.Request) {
    tables, err := db.ListTables(db.DB)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error listing tables: %v", err), http.StatusInternalServerError)
        return
    }

    // Return the tables as a JSON response
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(tables); err != nil {
        http.Error(w, fmt.Sprintf("Failed to encode tables: %v", err), http.StatusInternalServerError)
    }
}

func CreateTableAPI(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		TableName string `json:"table_name"`
	}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	sql, err := db.CreateTable(nil, requestBody.TableName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"sql": sql}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
	}
}