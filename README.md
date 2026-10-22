# api-service

This api-service is a service written to be used as a backend to a Postgres deployment.

It is primarily written with an o11y first guideline utilizing the [ex](https://github.com/circleci/ex) library and deployed via its [helm chart](https://github.com/imlogang/api-service/tree/main/deploy) to my local Kubernetes cluster.
Once there, the main consumer is a Discord bot I wrote with [DiscordJS](https://discord.js.org/).

O11y is observed through Honeycomb allowing tracing through the API endpoints.

## Database changes

Tables are managed with [goose](https://github.com/pressly/goose) migrations in `internal/db/migrations/`, which are embedded in the binary and applied when the service starts. To add or change a table:

1. Add a new file with the next number, e.g. `internal/db/migrations/00002_create_trivia_scores.sql`:
   ```sql
   -- +goose Up
   CREATE TABLE trivia_scores (
       id SERIAL PRIMARY KEY,
       username TEXT UNIQUE,
       score INTEGER NOT NULL DEFAULT 0
   );
   ```
2. Open a PR. CI runs every test against a fresh database with all migrations applied.
3. Merge. The next deploy applies it; applied migrations are recorded in the `goose_db_version` table.

Never edit a migration that has already been deployed; add a new one instead. Dropping a table is a migration too (`DROP TABLE ...`).

## Tests

Tests that need Postgres use [`dbfixture`](https://github.com/circleci/ex/tree/main/testing/dbfixture) to create a throwaway database per test. Point the `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER` and `POSTGRES_PASSWORD` environment variables at a Postgres you can create databases on; without one, those tests are skipped locally.
