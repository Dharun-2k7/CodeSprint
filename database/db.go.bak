package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// InitDB initializes the database connection
func InitDB() error {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "codesprint"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "codesprint123"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "codesprint"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize schema
	if err = InitSchema(); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

// InitSchema creates the database schema
func InitSchema() error {
	schema, err := os.ReadFile("database/schema.sql")
	if err != nil {
		// Try alternative path
		schema, err = os.ReadFile("./database/schema.sql")
		if err != nil {
			return fmt.Errorf("failed to read schema file: %w", err)
		}
	}

	_, err = DB.Exec(string(schema))
	if err != nil {
		// Ignore errors if tables already exist
		fmt.Printf("Schema initialization note: %v\n", err)
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

