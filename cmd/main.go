package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alecthomas/kong"
	"github.com/circleci/ex/httpserver"
	"github.com/circleci/ex/httpserver/healthcheck"
	"github.com/circleci/ex/termination"
	"github.com/imlogang/api-service/cmd/setup"
	"github.com/imlogang/api-service/internal/db"
	"github.com/imlogang/api-service/internal/internalapi"

	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/system"
)

var (
	Version = "dev"
	Date    = ""
)

type cli struct {
	APIAddr            string        `name:"api-addr" default:":8080" help:"Address for the internal API."`
	HealthcheckAPIAddr string        `name:"healthcheck-api-addr" default:":8081" help:"Address for the health check server."`
	ShutdownDelay      time.Duration `name:"shutdown-delay" default:"30s" help:"Shutdown delay."`
}

func main() {
	ctx := context.Background()
	err := run(ctx, Version, Date)
	if err != nil && !errors.Is(err, termination.ErrTerminated) {
		log.Fatal("Unexpected Error: ", err)
	}
}

func run(ctx context.Context, version, date string) (err error) {
	cli := cli{}
	kong.Parse(&cli)
	cfg := setup.O11ySetup()
	ctx, o11yCleanup, err := setup.LoadO11y(ctx, "internal-service", *cfg, version)
	if err != nil {
		log.Fatal(err)
	}
	defer o11yCleanup(ctx)

	ctx, runSpan := o11y.StartSpan(ctx, "main: run")
	defer o11y.End(runSpan, &err)

	sys := system.New()
	defer sys.Cleanup(ctx)

	store, err := loadDatabase(ctx, sys)
	if err != nil {
		return err
	}

	err = loadInternal(ctx, cli, sys, store)
	if err != nil {
		return err
	}

	o11yMessage := fmt.Sprintf("loading the healthchecks with gin on port: %s", cli.HealthcheckAPIAddr)
	o11y.Log(ctx, o11yMessage)
	_, err = healthcheck.Load(ctx, cli.HealthcheckAPIAddr, sys)
	if err != nil {
		return err
	}

	return sys.Run(ctx, 0)
}

func loadInternal(ctx context.Context, cli cli, sys *system.System, store *db.Store) error {
	a, err := httpapi.New(ctx, store)
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

func loadDatabase(ctx context.Context, sys *system.System) (store *db.Store, err error) {
	ctx, span := o11y.StartSpan(ctx, "Database Check")
	defer o11y.End(span, &err)

	store, err = db.New(ctx, db.LoadConfig())
	if err != nil {
		o11y.AddFieldToTrace(ctx, "db-check", fmt.Sprintf("database error: %s", err))
		o11y.AddFieldToTrace(ctx, "status", "unhealthy")
		return nil, err
	}
	sys.AddHealthCheck(store)
	sys.AddCleanup(func(ctx context.Context) error {
		store.Close()
		return nil
	})

	err = migrate(ctx, store)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "status", "migration_error")
		return nil, err
	}

	o11y.AddFieldToTrace(ctx, "db-check", "healthy")
	o11y.AddFieldToTrace(ctx, "status", "healthy")
	return store, nil
}

func migrate(ctx context.Context, store *db.Store) (err error) {
	ctx, span := o11y.StartSpan(ctx, "db.migrate")
	defer o11y.End(span, &err)

	applied, err := store.Migrate(ctx)
	if err != nil {
		return err
	}

	o11y.AddFieldToTrace(ctx, "migrations_applied", applied)
	o11y.AddFieldToTrace(ctx, "migrations_applied_count", len(applied))
	return nil
}
