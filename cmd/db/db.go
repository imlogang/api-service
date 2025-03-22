package db

import (
	"context"

	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx"
	_ "github.com/lib/pq"
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
	if conn == nil {
		return nil, fmt.Errorf("connection is nil")
	}

	if err != nil {
		log.Fatal("Error opening connection to the database:", err)
	}
	return conn, err
}

func ListTables() ([]string, error) {
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		log.Fatal("Error testing DB connection:", err)
	}
	var tableNames []string
	sql := `SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'`
	rows, err := DB.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %v", err)
	}
	defer DB.Close()

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

	if tableNames == nil {
		return nil, fmt.Errorf("there are no tables in the database")
	}
	return tableNames, nil
}

func CreateTable(tableName string) (string, error) {
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		log.Fatal("Error testing DB connection:", err)
	}
	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}

	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id SERIAL PRIMARY KEY, name TEXT);`, tableName)
	_, err = DB.Exec(sql)
	if err != nil {
		return "", fmt.Errorf(`there was an error creating the table:, %s`, err)
	}

	defer DB.Close()
	return fmt.Sprintf(`%s succesfully created.`, tableName), nil
}

func DeleteTable(tableName string) (string, error) {
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		log.Fatal("Error testing DB connection:", err)
	}
	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}
	sql := fmt.Sprintf(`DROP TABLE %s`, tableName)
	_, err = DB.Exec(sql)
	if err != nil {
		return "", fmt.Errorf(`there was an error creating the table:, %s`, err)
	}

	defer DB.Close()
	return fmt.Sprintf(`%s succesfully deleted.`, tableName), nil
}

func UpdateTableWithUser(tableName string, user string,) (string, error) {
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		log.Fatal("Error testing DB connection: ", err)
	}
	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}
	if user == "" {
		return "", fmt.Errorf("the user must not be empty")
	}
	sql := fmt.Sprintf(`INSERT INTO %s (USER, SCORE) VALUES (%s, 0)`, tableName, user)
	_, err = DB.Exec(sql)
	if err != nil {
		return "", fmt.Errorf(`there was an error updating the table: %s`, err)
	}
	defer DB.Close()
	return fmt.Sprintf("The table %s was updated", tableName), nil
}
