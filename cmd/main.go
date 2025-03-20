package main

import (
	"fmt"
	"go-api/cmd/api"
	"go-api/cmd/db"
	"log"
	"net/http"
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

	config := db.LoadConfig()
	err := config.TestDBConnection()
	if err != nil {
		log.Fatal("Error testing DB connection:", err)
		return
	} else {
		fmt.Println("Database connection succesfull.")
	}
	http.HandleFunc("/api/private/list_tables", httpapi.ListTablesAPI)
	http.HandleFunc("/api/private/create_table", httpapi.CreateTableAPI)

	// Start the server
	fmt.Println("Server started on http://localhost:8080")
	fmt.Println("You can also connect via http://go-api-service.go-api.svc.cluster.local:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}