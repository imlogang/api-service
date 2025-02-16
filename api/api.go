package httpapi

import (
	"encoding/json"
	"net/http"
	"go-api/deluge" 
)

type TorrentRequest struct {
	TorrentPath string `json:"torrentPath"`
}

// addTorrentHandler handles the HTTP request to add a torrent file
func AddTorrentHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the JSON request body
	var req TorrentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call the AuthAndDownloadTorrent function from the deluge package
	result, err := deluge.AuthAndDownloadTorrent(req.TorrentPath)
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