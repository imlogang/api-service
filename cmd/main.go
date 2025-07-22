package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/circleci/ex/httpserver"
	"github.com/circleci/ex/httpserver/healthcheck"
	"github.com/circleci/ex/termination"
	"go-api/cmd/api"
	_ "go-api/cmd/docs"
	"go-api/cmd/setup"
	"log"
	"net/http"
	"time"

	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/system"
)

type cli struct {
	APIAddr            string        `long:"api-addr" default:":8082" description:"api addr"`
	HealthcheckAPIAddr string        `long:"api-addr" default:":8081" description:"api addr for healthchecks"`
	ShutdownDelay      time.Duration `long:"shutdown-delay" default:"30s" description:"shutdown delay"`
}

// @title Logan's API
// @version 1.0
// @description These APIs handle a lot of backend things..
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email logan@logangodsey.com

// @host api-service.logangodsey.com
// @BasePath /api/private/
func main() {
	ctx := context.Background()
	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		log.Printf("error loading timezone: %s\n", err)
	}

	go func() {
		err := run(ctx, location)
		if err != nil && !errors.Is(err, termination.ErrTerminated) {
			log.Fatal("Unexpected Error: ", err)
		}
	}()

	apiHandler := httpapi.NewAPIHandler(ctx)

	http.HandleFunc("/api/private/add_torrent", httpapi.AddTorrentHandler)
	//http.HandleFunc("/api/private/hello", httpapi.HelloWorldHandler)
	//http.HandleFunc("/health", apiHandler.HealthCheckHandler)

	//http.HandleFunc("/api/private/list_tables", httpapi.ListTablesAPI)
	http.HandleFunc("/api/private/create_table", httpapi.CreateTableAPI)
	http.HandleFunc("/api/private/delete_table", httpapi.DeleteTableAPI)
	http.HandleFunc("/api/private/update_table_with_user", httpapi.UpdateTableWithUser)
	http.HandleFunc("/api/private/get_current_score", httpapi.GetScoreAPI)
	http.HandleFunc("/api/private/update_user_score", httpapi.UpdateScoreForUserAPI)
	http.HandleFunc("/api/private/get_pokemon", apiHandler.GetPokemonAPI)
	http.HandleFunc("/api/private/put_answer", httpapi.PutAnswerInDBAPI)
	http.HandleFunc("/api/private/get_answer", httpapi.ReadAnswerFromDBAPI)
	http.HandleFunc("/api/private/leaderboard", httpapi.LeaderboardAPI)

	//Start the server
	fmt.Println("Server started on http://localhost:8080")
	fmt.Println("You can also connect via http://go-api-service.go-api.svc.cluster.local:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}

func run(ctx context.Context, location *time.Location) (err error) {
	cli := cli{}
	kong.Parse(&cli)
	cfg := setup.O11ySetup()
	ctx, o11yCleanup, err := setup.LoadO11y(ctx, "api-service", *cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer o11yCleanup(ctx)

	ctx, runSpan := o11y.StartSpan(ctx, "main: run")
	defer o11y.End(runSpan, &err)

	o11y.Log(ctx, "starting api-service",
		o11y.Field("date", time.Now().In(location)),
	)
	sys := system.New()
	defer sys.Cleanup(ctx)

	err = loadInternal(ctx, cli, sys)
	if err != nil {
		return err
	}

	o11y.Log(ctx, "health checks are loaded",
		o11y.Field("date", time.Now().In(location)),
	)
	o11yMessage := fmt.Sprintf("loading the healthchecks with gin on port: %s", cli.HealthcheckAPIAddr)
	o11y.Log(ctx, o11yMessage)
	_, err = healthcheck.Load(ctx, cli.HealthcheckAPIAddr, sys)
	if err != nil {
		return err
	}

	return sys.Run(ctx, 0)
}

func loadInternal(ctx context.Context, cli cli, sys *system.System) error {
	a, err := httpapi.New(ctx)
	if err != nil {
		return err
	}

	o11yMessage := fmt.Sprintf("loading the httpserver with gin on port: %s", cli.APIAddr)
	o11y.Log(ctx, o11yMessage)

	_, err = httpserver.Load(ctx, httpserver.Config{
		Name:    "internalapi",
		Addr:    cli.APIAddr,
		Handler: a.Handler(),
	}, sys)

	o11y.Log(ctx, "all gin routes are ready")

	return err
}
