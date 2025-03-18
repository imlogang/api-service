package db

import (
	"context"

	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/"
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
	//connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", 
	//	c.Host, c.Port, c.User, c.Password, c.DB)
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.User, c.Password, c.Host, c.Port, c.DB)

	// Open a connection to the database
	//db, err := sql.Open("postgres", connStr)
	conn, err := pgx.Connect(context.Background(), connStr)
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

//nolint:unused,empty
func AddEntry() {
	return
}