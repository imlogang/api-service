package db

import (
	"context"

	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx"
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
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.User, c.Password, c.Host, c.Port, c.DB)

	connectionConfig, err := pgx.ParseConnectionString(connStr)
	if err != nil {
		log.Fatalf("Failed to parse connection string: %v\n", err)
	}
	conn, err := pgx.Connect(connectionConfig)
	if err != nil {
		log.Fatal("Error opening connection to the database:", err)
	}
	defer func(conn *pgx.Conn) {
		err := conn.Close()
		if err != nil {
			return
		}
	}(conn)

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
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

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

	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}

	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id SERIAL PRIMARY KEY, name TEXT);`, tableName)
	_, err = DB.Exec(sql)
	if err != nil {
		return "", fmt.Errorf(`there was an error creating the table:, %s`, err)
	}

	return fmt.Sprintf(`%s succesfully created.`, tableName), nil
}

func checkIfTableExists(tableName string) error {
	if tableName == "" {
		return fmt.Errorf("the table: %s must not be empty", tableName)
	}
	config := LoadConfig()
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
	sql := `SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = $1);`
	var exists bool
	err = DB.QueryRow(sql, tableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("there was an error checking if the table exists: %s", err)
	}
	if exists {
		return nil
	} else {
		_, err := CreateTable(tableName)
		if err != nil {
			return fmt.Errorf("there was an error creating your table: %s", err)
		}
	}
	return nil
}

func DeleteTable(tableName string) (string, error) {
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		log.Fatal("Error testing DB connection:", err)
	}

	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}
	sql := fmt.Sprintf(`DROP TABLE %s`, tableName)
	_, err = DB.Exec(sql)
	if err != nil {
		return "", fmt.Errorf(`there was an error creating the table:, %s`, err)
	}

	return fmt.Sprintf(`%s succesfully deleted.`, tableName), nil
}

func AddColumnsIfNotExists(tableName string) error {
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		return fmt.Errorf(`error testing DB connection: %s`, err)
	}
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

	sql_username := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS "USERNAME" VARCHAR(255);`, tableName)
	sql_score := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS "SCORE" INTEGER;`, tableName)

	_, err = DB.Exec(sql_username)
	if err != nil {
		return fmt.Errorf("error adding username columns: %s", err)
	}

	_, err = DB.Exec(sql_score)
	if err != nil {
		return fmt.Errorf("error adding score columns: %s", err)
	}

	return nil
}

func addColumnIfNotExistsAnswerTable(tableName string, column string, secondColumn string) error {
	if tableName == "" || column == "" || secondColumn == "" {
		return fmt.Errorf("tablename: %s, column: %s, or secondColumn: %s cannot be empty", tableName, column, secondColumn)
	}

	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		return fmt.Errorf(`error testing DB connection: %s`, err)
	}
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

	sql := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS "%s" VARCHAR(255);`, tableName, column)
	sqlSecond := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS "%s" INTEGER`, tableName, secondColumn)

	_, err = DB.Exec(sql)
	if err != nil {
		fmt.Printf("error adding %s columns: %s", column, err)
		return fmt.Errorf("error adding %s columns: %s", column, err)
	}
	_, err = DB.Exec(sqlSecond)
	if err != nil {
		fmt.Printf("error adding %s columns: %s", secondColumn, err)
		return fmt.Errorf("error adding %s columns: %w", secondColumn, err)
	}

	return nil
}

func AddUserIfNotExist(tableName string, username string) (string, error) {
	if tableName == "" || username == "" {
		return "", fmt.Errorf("tablename: %s, and username: %s, cannot be empty", tableName, username)
	}
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		log.Fatal("Error testing DB connection: ", err)
	}
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

	sql := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE "USERNAME" = $1;`, tableName)
	var exists int
	err = DB.QueryRow(sql, username).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("there was an error querying the database, %s", err)
	}

	if exists > 0 {
		return "the user exists", nil
	} else {
		response, err := UpdateTableWithUser(tableName, username)
		if err != nil {
			return "", fmt.Errorf("there was an error creating the user in the database, %s", err)
		}
		return response, nil
	}

}

func UpdateTableWithUser(tableName string, username string) (string, error) {
	err := AddColumnsIfNotExists(tableName)
	if err != nil {
		return "", fmt.Errorf("error ensuring columns: %v", err)
	}

	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		log.Fatal("Error testing DB connection: ", err)
	}

	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

	if tableName == "" {
		return "", fmt.Errorf("the table name must not be empty")
	}
	if username == "" {
		return "", fmt.Errorf("the user must not be empty")
	}
	sql := fmt.Sprintf(`INSERT INTO %s ("USERNAME", "SCORE") VALUES ('%s', 0)`, tableName, username)
	_, err = DB.Exec(sql)
	if err != nil {
		return "", fmt.Errorf(`there was an error updating the table: %s`, err)
	}

	return fmt.Sprintf("The table %s was updated", tableName), nil
}

func GetCurrentScore(tableName string, username string) (int, error) {
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		log.Fatal("Error testing DB connection: ", err)
	}
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)
	if tableName == "" || username == "" {
		return 0, fmt.Errorf("table or username must not be empty. table: %s, username: %s", tableName, username)
	}
	sql := fmt.Sprintf(`SELECT "SCORE" FROM "%s" WHERE "USERNAME" = $1;`, tableName)
	var score int
	err = DB.QueryRow(sql, username).Scan(&score)
	if err != nil {
		if err == pgx.ErrNoRows {
			_, err := AddUserIfNotExist(tableName, username)
			if err != nil {
				return 0, fmt.Errorf("there was an error creating your user, %s", err)
			}
			return 0, nil
		}
		return 0, fmt.Errorf("there was an error finding the score for a username. %s", err)
	}
	return score, nil
}

func UpdateScoreForUser(tableName string, username string, score int, column string) (string, error) {
	if tableName == "" || username == "" || score == 0 || column == "" {
		return "", fmt.Errorf("tablename: %s, username: %s, score: %d, or column: %s must not be empty", tableName, username, score, column)
	}
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		log.Fatal("Error testing DB connection: ", err)
	}
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)
	sql := fmt.Sprintf(`UPDATE %s SET "%s" = %d WHERE "USERNAME" = '%s'`, tableName, column, score, username)
	_, err = DB.Exec(sql)
	if err != nil {
		return "", fmt.Errorf("there was an error updating the users score. %s", err)
	}

	return "the score for the user has been updated", nil
}

func PutAnswerInDB(tablenName string, answer string, column string, secondColumn string, numberInArray int) (string, error) {
	if tablenName == "" || answer == "" || column == "" || secondColumn == "" {
		return "", fmt.Errorf("the tablename: %s, answer: %s, column: %s, numberInArray: %d, or secondColumn: %s cannot be empty", tablenName, answer, column, numberInArray, secondColumn)
	}

	err := checkIfTableExists(tablenName)
	if err != nil {
		return "", fmt.Errorf("there was an error creating your table: %s", err)
	}
	err = addColumnIfNotExistsAnswerTable(tablenName, column, secondColumn)
	if err != nil {
		return "", fmt.Errorf("there was an error adding your column: %s", column)
	}

	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		return "", fmt.Errorf("there was an error connecting to the database: %s", err)
	}
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)
	sql := fmt.Sprintf(`INSERT INTO %s ("id", "ANSWER", "POSITION") VALUES (1, $1, $2) ON CONFLICT ("id") DO UPDATE SET "ANSWER" = $1, "POSITION" = $2;`, tablenName)
	_, err = DB.Exec(sql, answer, numberInArray)
	if err != nil {
		return "", fmt.Errorf("there was an error updating/creating the row: %s", err)
	}
	return fmt.Sprintf("the %s table has been updated with %s", tablenName, answer), nil
}

func ReadAnswerFromDB(tableName string, column string) (string, error) {
	if tableName == "" || column == "" {
		return "", fmt.Errorf("tablename: %s or column: %s cannot be empty", tableName, column)
	}
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		return "", fmt.Errorf("there was an error connecting to the database: %s", err)
	}
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

	sql := fmt.Sprintf("SELECT '%s' FROM %s WHERE id = 1", column, tableName)
	var answer string
	err = DB.QueryRow(sql).Scan(&answer)
	if err != nil {
		return "", fmt.Errorf("there was an error finding the answer: %s", err)
	}
	return answer, nil
}

func GetLeaderboard(tableName string) (string, error) {
	if tableName == "" {
		return "", fmt.Errorf("tablename: %s", tableName)
	}
	config := LoadConfig()
	DB, err := config.Connect()
	if err != nil {
		return "", fmt.Errorf("there was an error connecting to the database: %s", err)
	}
	defer func(DB *pgx.Conn) {
		err := DB.Close()
		if err != nil {
			return
		}
	}(DB)

	sql := fmt.Sprintf(`SELECT "USERNAME", "SCORE" FROM %s ORDER BY "SCORE" DESC LIMIT 10;`, tableName)
	rows, err := DB.Query(sql)
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
