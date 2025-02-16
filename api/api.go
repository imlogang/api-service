package httpapi

import (
	"encoding/json"
	"net/http"
	"go-api/deluge" 
	"log"
	"io"
)

type TorrentRequest struct {
	Parameters struct {
		URL string `json:"url"`
	} `json:"parameters"`
}

// addTorrentHandler handles the HTTP request to add a torrent file
func AddTorrentHandler(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusInternalServerError)
		return
	}

	// Log the raw request body for debugging purposes
	log.Printf("Received request body: %s", string(bodyBytes))

	// Initialize the struct to hold the parsed request
	var req TorrentRequest

	// Parse the JSON request body into the struct
	err = json.NewDecoder(r.Body).Decode(&req)
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

	// Return the result as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World!"))
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// You can add more complex logic here, like checking DB or external services.
	w.WriteHeader(http.StatusOK)
}