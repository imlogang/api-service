package main
import (
	"log"
	"net/http"
	"deluge-api/api"
)

func main() {
	// Define the route for adding a torrent
	http.HandleFunc("/add_torrent", httpapi.AddTorrentHandler)
	http.HandleFunc("/hello", httpapi.HelloWorldHandler)

	// Start the server
	log.Fatal(http.ListenAndServe(":8080", nil))
}