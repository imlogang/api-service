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
	"github.com/imlogang/api-service/cmd/db"
	"github.com/imlogang/api-service/cmd/internal"
	"github.com/imlogang/api-service/cmd/setup"
	"github.com/jackc/pgx"

	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/system"
)

type cli struct {
	APIAddr            string        `long:"internal-addr" default:":8080" description:"internal addr"`
	HealthcheckAPIAddr string        `long:"internal-addr" default:":8081" description:"internal addr for healthchecks"`
	ShutdownDelay      time.Duration `long:"shutdown-delay" default:"30s" description:"shutdown delay"`
}

func main() {
	ctx := context.Background()
	err := run(ctx)
	if err != nil && !errors.Is(err, termination.ErrTerminated) {
		log.Fatal("Unexpected Error: ", err)
	}
}

func run(ctx context.Context) (err error) {
	cli := cli{}
	kong.Parse(&cli)
	cfg := setup.O11ySetup()
	ctx, o11yCleanup, err := setup.LoadO11y(ctx, "internal-service", *cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer o11yCleanup(ctx)

	testDatabase(ctx)

	ctx, runSpan := o11y.StartSpan(ctx, "main: run")
	defer o11y.End(runSpan, &err)

	sys := system.New()
	defer sys.Cleanup(ctx)

	err = loadInternal(ctx, cli, sys)
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
	_, err := config.TestDBConnection()
	if err != nil {
		databaseError := fmt.Sprintf("database error: %s", err)
		o11y.AddFieldToTrace(ctx, "db-check", databaseError)
		o11y.AddFieldToTrace(ctx, "status", "unhealthy")
		return
	}

	err = ensurePokemonScoresTable(ctx)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "status", "schema_error")
		o11y.AddFieldToTrace(ctx, "error", err.Error())
		return
	}

	o11y.AddFieldToTrace(ctx, "db-check", "healthy")
	o11y.AddFieldToTrace(ctx, "status", "healthy")
}

func ensurePokemonScoresTable(ctx context.Context) (err error) {
	config := db.LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		return err
	}
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

	ctx, span := o11y.StartSpan(ctx, "db.ensure_pokemon_scores")
	defer o11y.End(span, &err)

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS pokemon_scores (
			id SERIAL PRIMARY KEY
		);

		ALTER TABLE pokemon_scores
			ADD COLUMN IF NOT EXISTS username TEXT UNIQUE,
			ADD COLUMN IF NOT EXISTS score INTEGER NOT NULL DEFAULT 0;
	`)
	if err != nil {
		o11y.AddFieldToTrace(ctx, "table", "pokemon_scores")
		o11y.AddFieldToTrace(ctx, "error", err.Error())
		return err
	}

	o11y.AddFieldToTrace(ctx, "table", "pokemon_scores")
	o11y.AddFieldToTrace(ctx, "status", "ensured")
	return nil
}
