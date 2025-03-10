package main
import (
	"fmt"
	"log"
	"net/http"
	"go-api/api"
)

func main() {
	// Define the route for adding a torrent
	http.HandleFunc("/api/private/add_torrent", httpapi.AddTorrentHandler)
	http.HandleFunc("/api/private/hello", httpapi.HelloWorldHandler)
	http.HandleFunc("/health", httpapi.HealthCheckHandler)
	http.HandleFunc("/api/private/root", httpapi.GetRoot)
	http.HandleFunc("/resume", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./website/resume.html")
	})
	http.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./website/blog/blog.html")
	})
	http.Handle("/", http.FileServer(http.Dir("./website")))

	// Start the server
	fmt.Println("Server started on http://localhost:8080")
	fmt.Println("You can also connect via http://go-api-service.go-api.svc.cluster.local:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}