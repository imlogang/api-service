package httpapi

import (
	"encoding/json"
	"net/http"
	"go-api/deluge" 
)

// addTorrentHandler handles the HTTP request to add a torrent file
func AddTorrentHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for torrent magnet link
	torrentPath := r.URL.Query().Get("torrentPath")

	if torrentPath == "" {
		http.Error(w, "Torrent path is required", http.StatusBadRequest)
		return
	}

	// Call the AuthAndDownloadTorrent function from the deluge package
	result, err := deluge.AuthAndDownloadTorrent(torrentPath)
	if err != nil {
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