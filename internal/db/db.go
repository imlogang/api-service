package db

import (
	"context"
	"errors"

	"fmt"
	"net"
	"net/url"
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

// connString builds the URL by parts so that characters like @, / or : in
// the password or database name are escaped rather than read as separators.
func (c *Config) connString() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, c.Port),
		Path:   "/" + c.DB,
	}
	return u.String()
}

// ErrColumnNotUpdatable is returned when a caller tries to update a column
// other than the ones in updatableColumns.
var ErrColumnNotUpdatable = errors.New("column cannot be updated")

// ErrUserNotFound is returned when updating a user that isn't in the table.
var ErrUserNotFound = errors.New("user not found")

// ErrNegativeScore is returned when setting a score below zero.
var ErrNegativeScore = errors.New("score must not be negative")

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

func (s *Store) GetCurrentScore(tableName string, username string, ctx context.Context) (int, error) {
	if tableName == "" || username == "" {
		return 0, fmt.Errorf("table or username must not be empty. table: %s, username: %s", tableName, username)
	}
	sql := fmt.Sprintf(`SELECT "score" FROM %s WHERE "username" = $1;`, ident(tableName))
	var score int
	err := s.pool.QueryRow(ctx, sql, username).Scan(&score)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err := s.createUser(ctx, tableName, username)
			if err != nil {
				return 0, fmt.Errorf("there was an error creating your user, %s", err)
			}
			return 0, nil
		}
		return 0, fmt.Errorf("there was an error finding the score for a username. %s", err)
	}
	return score, nil
}

// createUser adds the user with a score of 0. It does nothing if the user
// already exists, so two requests creating the same new user can't collide.
func (s *Store) createUser(ctx context.Context, tableName string, username string) error {
	sql := fmt.Sprintf(`INSERT INTO %s ("username", "score") VALUES ($1, 0)
		ON CONFLICT ("username") DO NOTHING`, ident(tableName))
	_, err := s.pool.Exec(ctx, sql, username)
	return err
}

// UpdateScoreForUser sets a user's score, for example to correct or reset it.
func (s *Store) UpdateScoreForUser(tableName string, username string, score int, column string, ctx context.Context) (string, error) {
	if tableName == "" || username == "" || column == "" {
		return "", fmt.Errorf("tablename: %s, username: %s, or column: %s must not be empty", tableName, username, column)
	}
	if !updatableColumns[column] {
		return "", fmt.Errorf("%w: %s", ErrColumnNotUpdatable, column)
	}
	if score < 0 {
		return "", fmt.Errorf("%w: %d", ErrNegativeScore, score)
	}
	sql := fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE "username" = $2`, ident(tableName), ident(column))
	tag, err := s.pool.Exec(ctx, sql, score, username)
	if err != nil {
		return "", fmt.Errorf("there was an error updating the users score. %s", err)
	}
	if tag.RowsAffected() == 0 {
		return "", fmt.Errorf("%w: %s", ErrUserNotFound, username)
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

// Standing is where a user places in a table.
type Standing struct {
	// Rank is 1 for the top score. Equal scores share a rank, and the next
	// rank skips ahead: 9, 9, 7 ranks as 1, 1, 3.
	Rank int
	// Players is how many users the table has, including those on 0.
	Players int
}

// GetStanding returns the user's rank in the table and how many players it has.
func (s *Store) GetStanding(ctx context.Context, tableName string, username string) (Standing, error) {
	if tableName == "" || username == "" {
		return Standing{}, fmt.Errorf("tablename: %s and username: %s must not be empty", tableName, username)
	}
	sql := fmt.Sprintf(`SELECT "rank", "players" FROM (
			SELECT "username",
				RANK() OVER (ORDER BY "score" DESC) AS "rank",
				COUNT(*) OVER () AS "players"
			FROM %s
		) AS ranked
		WHERE "username" = $1`, ident(tableName))
	var st Standing
	err := s.pool.QueryRow(ctx, sql, username).Scan(&st.Rank, &st.Players)
	if errors.Is(err, pgx.ErrNoRows) {
		return Standing{}, fmt.Errorf("%w: %s", ErrUserNotFound, username)
	}
	if err != nil {
		return Standing{}, fmt.Errorf("there was an error finding the users rank. %s", err)
	}
	return st, nil
}

type LeaderboardEntry struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
}

// GetLeaderboard returns the top 10 users by score, highest first. Equal
// scores are ordered by username so the list and the cutoff are stable.
func (s *Store) GetLeaderboard(tableName string, ctx context.Context) ([]LeaderboardEntry, error) {
	if tableName == "" {
		return nil, fmt.Errorf("tablename: %s", tableName)
	}

	sql := fmt.Sprintf(`SELECT "username", "score" FROM %s ORDER BY "score" DESC, "username" LIMIT 10;`, ident(tableName))
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
