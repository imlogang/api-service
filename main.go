package main
import (
	"fmt"
	"log"
	"net/http"
	"go-api/api"
)

func main() {
	// Define the route for adding a torrent
	http.HandleFunc("/add_torrent", httpapi.AddTorrentHandler)
	http.HandleFunc("/hello", httpapi.HelloWorldHandler)
	http.HandleFunc("/health", httpapi.HealthCheckHandler)
	http.HandleFunc("/root", httpapi.GetRoot)
	http.Handle("/", http.FileServer(http.Dir("./website")))

	// Start the server
	fmt.Println("Server started on http://localhost:8080")
	fmt.Println("You can also connect via http://go-api-service.go-api.svc.cluster.local:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}