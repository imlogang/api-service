package main

import (
	"fmt"
	"go-api/cmd/api"
	"go-api/cmd/db"
	_ "go-api/cmd/docs"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/swaggo/http-swagger"
)

// @title Logan's API
// @version 1.0
// @description These APIs handle a lot of backend things..
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email logan@logangodsey.com

// @host api-service.logangodsey.com
// @BasePath /api/private/
func main() {
	r := chi.NewRouter()
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))
	http.Handle("/docs/swagger.json", http.StripPrefix("/docs", http.FileServer(http.Dir("./cmd/docs"))))
	http.HandleFunc("/api/private/add_torrent", httpapi.AddTorrentHandler)
	http.HandleFunc("/api/private/hello", httpapi.HelloWorldHandler)
	http.HandleFunc("/health", httpapi.HealthCheckHandler)
	http.HandleFunc("/api/private/root", httpapi.GetRoot)
	http.HandleFunc("/resume", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./cmd/website/resume.html")
	})
	http.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./cmd/website/blog/blog.html")
	})
	http.Handle("/", http.FileServer(http.Dir("./cmd/website")))

	config := db.LoadConfig()
	err := config.TestDBConnection()
	if err != nil {
		log.Fatal("Error testing DB connection:", err)
		return
	}

	http.HandleFunc("/api/private/list_tables", httpapi.ListTablesAPI)
	http.HandleFunc("/api/private/create_table", httpapi.CreateTableAPI)
	http.HandleFunc("/api/private/delete_table", httpapi.DeleteTableAPI)
	http.HandleFunc("/api/private/update_table_with_user", httpapi.UpdateTableWithUser)

	// Start the server
	fmt.Println("Server started on http://localhost:8080")
	fmt.Println("You can also connect via http://go-api-service.go-api.svc.cluster.local:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
