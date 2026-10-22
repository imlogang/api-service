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
	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}
	if username == "" {
		return "", fmt.Errorf("the user must not be empty")
	}
	sql := fmt.Sprintf(`INSERT INTO %s ("username", "score") VALUES ($1, 0)`, ident(tableName))
	_, err := s.pool.Exec(ctx, sql, username)
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

// IncrementScore adds one point to the user's score in a single statement,
// creating the user if needed, and returns the new score.
func (s *Store) IncrementScore(ctx context.Context, tableName string, username string) (int, error) {
	if tableName == "" || username == "" {
		return 0, fmt.Errorf("tablename: %s and username: %s must not be empty", tableName, username)
	}
	sql := fmt.Sprintf(`INSERT INTO %s AS t ("username", "score") VALUES ($1, 1)
		ON CONFLICT ("username") DO UPDATE SET "score" = COALESCE(t."score", 0) + 1
		RETURNING "score"`, ident(tableName))
	var score int
	err := s.pool.QueryRow(ctx, sql, username).Scan(&score)
	if err != nil {
		return 0, fmt.Errorf("there was an error incrementing the users score. %s", err)
	}
	return score, nil
}

type LeaderboardEntry struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
}

// GetLeaderboard returns the top 10 users by score, highest first.
func (s *Store) GetLeaderboard(tableName string, ctx context.Context) ([]LeaderboardEntry, error) {
	if tableName == "" {
		return nil, fmt.Errorf("tablename: %s", tableName)
	}

	sql := fmt.Sprintf(`SELECT "username", "score" FROM %s ORDER BY "score" DESC LIMIT 10;`, ident(tableName))
	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %s", err)
	}

	leaderboard, err := pgx.CollectRows(rows, pgx.RowToStructByPos[LeaderboardEntry])
	if err != nil {
		return nil, fmt.Errorf("error reading leaderboard rows: %s", err)
	}
	return leaderboard, nil
}
