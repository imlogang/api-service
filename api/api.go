package httpapi

import (
	"log"
	"encoding/json"
	"net/http"
	"go-api/deluge" 
)

// addTorrentHandler handles the HTTP request to add a torrent file
func AddTorrentHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for torrent file path and download directory
	torrentPath := r.URL.Query().Get("torrentPath")

	// Call ConnectToDeluge to ensure we are connected before proceeding
	err := deluge.AddHostAndConnect()
	if err != nil {
		log.Printf("Error connecting to Deluge: %v", err)
		http.Error(w, "Error connecting to Deluge: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Call the AddTorrentFile function from the deluge package
	result, err := deluge.AddTorrentFile(torrentPath)
	if err != nil {
		log.Printf("Error adding torrent: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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