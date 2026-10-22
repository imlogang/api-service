package db

import (
	"context"
	"errors"

	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

func LoadConfig() Config {
	return Config{
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     os.Getenv("POSTGRES_PORT"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		DB:       os.Getenv("POSTGRES_DB"),
	}
}

func (c *Config) connString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.User, c.Password, c.Host, c.Port, c.DB)
}

// ErrColumnNotUpdatable is returned when a caller tries to update a column
// other than the ones in updatableColumns.
var ErrColumnNotUpdatable = errors.New("column cannot be updated")

var updatableColumns = map[string]bool{"score": true}

// ident quotes a table or column name so it can be safely placed in SQL.
// Postgres can't take identifiers as $N parameters, so this is used instead.
func ident(name string) string {
	return pgx.Identifier{name}.Sanitize()
}

// Store holds a single connection pool that is shared by every request.
type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, c Config) (*Store, error) {
	poolConfig, err := pgxpool.ParseConfig(c.connString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}
	poolConfig.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating the database pool: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("error connecting to the database: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

// HealthChecks marks the service not ready while the database is unreachable.
// Liveness is left nil so a database outage doesn't restart the pod.
func (s *Store) HealthChecks() (name string, ready, live func(ctx context.Context) error) {
	return "postgres", s.pool.Ping, nil
}

// Schema is applied at startup, and by the tests to each fresh database.
const Schema = `
	CREATE TABLE IF NOT EXISTS pokemon_scores (
		id SERIAL PRIMARY KEY
	);

	ALTER TABLE pokemon_scores
		ADD COLUMN IF NOT EXISTS username TEXT UNIQUE,
		ADD COLUMN IF NOT EXISTS score INTEGER NOT NULL DEFAULT 0;
`

func (s *Store) EnsurePokemonScoresTable(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, Schema)
	return err
}

func (s *Store) ListTables(ctx context.Context) ([]string, error) {
	sql := `SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'`
	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %v", err)
	}

	tableNames, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("error iterating over rows: %v", err)
	}

	if len(tableNames) == 0 {
		return nil, fmt.Errorf("there are no tables in the database")
	}
	return tableNames, nil
}

func (s *Store) CreateTable(tableName string, ctx context.Context) (string, error) {
	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}

	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id SERIAL PRIMARY KEY, name TEXT);`, ident(tableName))
	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return "", fmt.Errorf(`there was an error creating the table:, %s`, err)
	}

	return fmt.Sprintf(`%s succesfully created.`, tableName), nil
}

func (s *Store) checkIfTableExists(tableName string) error {
	if tableName == "" {
		return fmt.Errorf("the table: %s must not be empty", tableName)
	}
	sql := `SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = $1);`
	var exists bool
	err := s.pool.QueryRow(context.Background(), sql, tableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("there was an error checking if the table exists: %s", err)
	}
	if exists {
		return nil
	} else {
		_, err := s.CreateTable(tableName, context.Background())
		if err != nil {
			return fmt.Errorf("there was an error creating your table: %s", err)
		}
	}
	return nil
}

func (s *Store) DeleteTable(tableName string, ctx context.Context) (string, error) {
	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}
	sql := fmt.Sprintf(`DROP TABLE %s`, ident(tableName))
	_, err := s.pool.Exec(ctx, sql)
	if err != nil {
		return "", fmt.Errorf(`there was an error creating the table:, %s`, err)
	}

	return fmt.Sprintf(`%s succesfully deleted.`, tableName), nil
}

func (s *Store) AddColumnsIfNotExists(tableName string, ctx context.Context) error {
	sql_username := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS "username" VARCHAR(255);`, ident(tableName))
	sql_score := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS "score" INTEGER;`, ident(tableName))

	_, err := s.pool.Exec(ctx, sql_username)
	if err != nil {
		return fmt.Errorf("error adding username columns: %s", err)
	}

	_, err = s.pool.Exec(ctx, sql_score)
	if err != nil {
		return fmt.Errorf("error adding score columns: %s", err)
	}

	return nil
}

func (s *Store) addColumnIfNotExistsAnswerTable(tableName string, column string, secondColumn string) error {
	if tableName == "" || column == "" || secondColumn == "" {
		return fmt.Errorf("tablename: %s, column: %s, or secondColumn: %s cannot be empty", tableName, column, secondColumn)
	}

	sql := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s VARCHAR(255);`, ident(tableName), ident(column))
	sqlSecond := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s INTEGER`, ident(tableName), ident(secondColumn))

	_, err := s.pool.Exec(context.Background(), sql)
	if err != nil {
		fmt.Printf("error adding %s columns: %s", column, err)
		return fmt.Errorf("error adding %s columns: %s", column, err)
	}
	_, err = s.pool.Exec(context.Background(), sqlSecond)
	if err != nil {
		fmt.Printf("error adding %s columns: %s", secondColumn, err)
		return fmt.Errorf("error adding %s columns: %w", secondColumn, err)
	}

	return nil
}

func (s *Store) AddUserIfNotExist(tableName string, username string, ctx context.Context) (string, error) {
	if tableName == "" || username == "" {
		return "", fmt.Errorf("tablename: %s, and username: %s, cannot be empty", tableName, username)
	}

	sql := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE "username" = $1;`, ident(tableName))
	var exists int
	err := s.pool.QueryRow(ctx, sql, username).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("there was an error querying the database, %s", err)
	}

	if exists > 0 {
		return "the user exists", nil
	} else {
		response, err := s.UpdateTableWithUser(tableName, username, ctx)
		if err != nil {
			return "", fmt.Errorf("there was an error creating the user in the database, %s", err)
		}
		return response, nil
	}

}

func (s *Store) UpdateTableWithUser(tableName string, username string, ctx context.Context) (string, error) {
	err := s.AddColumnsIfNotExists(tableName, ctx)
	if err != nil {
		return "", fmt.Errorf("error ensuring columns: %v", err)
	}

	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}
	if username == "" {
		return "", fmt.Errorf("the user must not be empty")
	}
	sql := fmt.Sprintf(`INSERT INTO %s ("username", "score") VALUES ($1, 0)`, ident(tableName))
	_, err = s.pool.Exec(ctx, sql, username)
	if err != nil {
		return "", fmt.Errorf(`there was an error updating the table: %s`, err)
	}

	return fmt.Sprintf("The table %s was updated", tableName), nil
}

func (s *Store) GetCurrentScore(tableName string, username string, ctx context.Context) (int, error) {
	if tableName == "" || username == "" {
		return 0, fmt.Errorf("table or username must not be empty. table: %s, username: %s", tableName, username)
	}
	sql := fmt.Sprintf(`SELECT "score" FROM %s WHERE "username" = $1;`, ident(tableName))
	var score int
	err := s.pool.QueryRow(ctx, sql, username).Scan(&score)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_, err := s.AddUserIfNotExist(tableName, username, ctx)
			if err != nil {
				return 0, fmt.Errorf("there was an error creating your user, %s", err)
			}
			return 0, nil
		}
		return 0, fmt.Errorf("there was an error finding the score for a username. %s", err)
	}
	return score, nil
}

func (s *Store) UpdateScoreForUser(tableName string, username string, score int, column string, ctx context.Context) (string, error) {
	if tableName == "" || username == "" || score == 0 || column == "" {
		return "", fmt.Errorf("tablename: %s, username: %s, score: %d, or column: %s must not be empty", tableName, username, score, column)
	}
	if !updatableColumns[column] {
		return "", fmt.Errorf("%w: %s", ErrColumnNotUpdatable, column)
	}
	sql := fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE "username" = $2`, ident(tableName), ident(column))
	_, err := s.pool.Exec(ctx, sql, score, username)
	if err != nil {
		return "", fmt.Errorf("there was an error updating the users score. %s", err)
	}

	return "the score for the user has been updated", nil
}

func (s *Store) PutAnswerInDB(tablenName string, answer string, column string, secondColumn string, numberInArray int) (string, error) {
	if tablenName == "" || answer == "" || column == "" || secondColumn == "" {
		return "", fmt.Errorf("the tablename: %s, answer: %s, column: %s, numberInArray: %d, or secondColumn: %s cannot be empty", tablenName, answer, column, numberInArray, secondColumn)
	}

	err := s.checkIfTableExists(tablenName)
	if err != nil {
		return "", fmt.Errorf("there was an error creating your table: %s", err)
	}
	err = s.addColumnIfNotExistsAnswerTable(tablenName, column, secondColumn)
	if err != nil {
		return "", fmt.Errorf("there was an error adding your column: %s", column)
	}

	sql := fmt.Sprintf(`INSERT INTO %s ("id", "ANSWER", "POSITION") VALUES (1, $1, $2) ON CONFLICT ("id") DO UPDATE SET "ANSWER" = $1, "POSITION" = $2;`, ident(tablenName))
	_, err = s.pool.Exec(context.Background(), sql, answer, numberInArray)
	if err != nil {
		return "", fmt.Errorf("there was an error updating/creating the row: %s", err)
	}
	return fmt.Sprintf("the %s table has been updated with %s", tablenName, answer), nil
}

func (s *Store) ReadAnswerFromDB(tableName string, column string, ctx context.Context) (string, error) {
	if tableName == "" || column == "" {
		return "", fmt.Errorf("tablename: %s or column: %s cannot be empty", tableName, column)
	}

	sql := fmt.Sprintf("SELECT %s::text FROM %s WHERE id = 1", ident(column), ident(tableName))
	var answer string
	err := s.pool.QueryRow(ctx, sql).Scan(&answer)
	if err != nil {
		return "", fmt.Errorf("there was an error finding the answer: %s", err)
	}
	return answer, nil
}

func (s *Store) GetLeaderboard(tableName string, ctx context.Context) (string, error) {
	if tableName == "" {
		return "", fmt.Errorf("tablename: %s", tableName)
	}

	sql := fmt.Sprintf(`SELECT "username", "score" FROM %s ORDER BY "score" DESC LIMIT 10;`, ident(tableName))
	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return "", fmt.Errorf("error executing query: %s", err)
	}
	defer rows.Close()

	var leaderboard string
	for rows.Next() {
		var username string
		var score int
		err := rows.Scan(&username, &score)
		if err != nil {
			return "", fmt.Errorf("error scanning row: %s", err)
		}
		leaderboard += fmt.Sprintf("Username: %s, Score: %d\n", username, score)
	}

	if err = rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating through rows: %s", err)
	}

	if leaderboard == "" {
		return "No leaderboard data found.", nil
	}

	return leaderboard, nil
}
