package db

import (
	"context"

	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx"
	_ "github.com/lib/pq"
)

var DB *pgx.Conn
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

func (c *Config) TestDBConnection() error {
	// Connection string
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.User, c.Password, c.Host, c.Port, c.DB)

	// Open a connection to the database
	connectionConfig, err := pgx.ParseConnectionString(connStr)
	if err != nil {
		log.Fatalf("Failed to parse connection string: %v\n", err)
	}
	conn, err := pgx.Connect(connectionConfig)
	if err != nil {
		log.Fatal("Error opening connection to the database:", err)
	}
	defer conn.Close()

	// Test the connection
	err = conn.Ping(context.Background())
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	} else {
		fmt.Println("Successfully connected to the database!")
	}
	return err
}

func (c *Config) Connect() (*pgx.Conn, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.User, c.Password, c.Host, c.Port, c.DB)
	connectionConfig, err := pgx.ParseConnectionString(connStr)
	if err != nil {
		log.Fatalf("Failed to parse connection string: %v\n", err)
	}
	conn, err := pgx.Connect(connectionConfig)
	if err != nil {
		log.Fatal("Error opening connection to the database:", err)
	}
	return conn, nil
}

func ListTables(conn *pgx.Conn) ([]string, error) {
    var tableNames []string
    sql := `SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'`
    rows, err := conn.Query(sql)
    if err != nil {
        return nil, fmt.Errorf("failed to query tables: %v", err)
    }
    defer rows.Close()

    for rows.Next() {
        var tableName string
        err := rows.Scan(&tableName)
        if err != nil {
            return nil, fmt.Errorf("failed to scan row: %v", err)
        }
        tableNames = append(tableNames, tableName)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating over rows: %v", err)
    }

    return tableNames, nil
}