package main

import (
	"context"
	"fmt"
	"github.com/circleci/ex/o11y"
	"go-api/cmd/api"
	"go-api/cmd/db"
	_ "go-api/cmd/docs"
	"go-api/cmd/setup"
	"log"
	"net/http"
	"time"
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
	cfg := setup.O11ySetup()
	ctx := context.Background()
	ctx, o11yCleanup, err := setup.LoadO11y(ctx, "ap-service", *cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer o11yCleanup(ctx)

	ctx, runSpan := o11y.StartSpan(ctx, "main: run")
	defer o11y.End(runSpan, &err)

	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		log.Printf("error loading timezone: %s\n", err)
	}
	o11y.Log(ctx, "starting api-service",
		o11y.Field("date", time.Now().In(location)),
	)

	apiHandler := httpapi.NewAPIHandler(ctx)

	http.HandleFunc("/api/private/add_torrent", httpapi.AddTorrentHandler)
	http.HandleFunc("/api/private/hello", httpapi.HelloWorldHandler)
	http.HandleFunc("/health", httpapi.HealthCheckHandler)

	http.HandleFunc("/api/private/list_tables", httpapi.ListTablesAPI)
	http.HandleFunc("/api/private/create_table", httpapi.CreateTableAPI)
	http.HandleFunc("/api/private/delete_table", httpapi.DeleteTableAPI)
	http.HandleFunc("/api/private/update_table_with_user", httpapi.UpdateTableWithUser)
	http.HandleFunc("/api/private/get_current_score", httpapi.GetScoreAPI)
	http.HandleFunc("/api/private/update_user_score", httpapi.UpdateScoreForUserAPI)
	http.HandleFunc("/api/private/get_pokemon", apiHandler.GetPokemonAPI)
	http.HandleFunc("/api/private/put_answer", httpapi.PutAnswerInDBAPI)
	http.HandleFunc("/api/private/get_answer", httpapi.ReadAnswerFromDBAPI)
	http.HandleFunc("/api/private/leaderboard", httpapi.LeaderboardAPI)

	// Start the server
	fmt.Println("Server started on http://localhost:8080")
	fmt.Println("You can also connect via http://go-api-service.go-api.svc.cluster.local:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
