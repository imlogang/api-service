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
	"go-api/cmd/db"
	"go-api/cmd/setup"
	"log"
	"time"

	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/system"
)

type cli struct {
	APIAddr            string        `long:"api-addr" default:":8080" description:"api addr"`
	HealthcheckAPIAddr string        `long:"api-addr" default:":8081" description:"api addr for healthchecks"`
	ShutdownDelay      time.Duration `long:"shutdown-delay" default:"30s" description:"shutdown delay"`
}

func main() {
	ctx := context.Background()
	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		log.Printf("error loading timezone: %s\n", err)
	}

	err = run(ctx, location)
	if err != nil && !errors.Is(err, termination.ErrTerminated) {
		log.Fatal("Unexpected Error: ", err)
	}
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

	testDatabase(ctx)

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

func testDatabase(ctx context.Context) {
	ctx, span := o11y.StartSpan(ctx, "Database Check")
	defer span.End()

	config := db.LoadConfig()
	err := config.TestDBConnection()
	if err != nil {
		databaseError := fmt.Sprintf("database error: %s", err)
		o11y.AddFieldToTrace(ctx, "db-check", databaseError)
		o11y.AddFieldToTrace(ctx, "status", "unhealthy")
		return
	}

	o11y.AddFieldToTrace(ctx, "db-check", "healthy")
	o11y.AddFieldToTrace(ctx, "status", "healthy")
}
