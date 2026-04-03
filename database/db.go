package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"

	_ "embed"
)

var DB *sql.DB

// Embed SQL so the app can start reliably on managed platforms
// where the working directory may differ.
//
//go:embed schema.sql
var schemaSQL string

//go:embed migrations.sql
var migrationsSQL string

// InitDB initializes the database connection
func InitDB() error {
	// Preferred: Neon/managed Postgres via connection string.
	// Example: postgres://user:pass@host:5432/dbname?sslmode=require
	connStr := os.Getenv("DB_URL")

	// Back-compat: allow legacy envs when DB_URL isn't provided.
	if connStr == "" {
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

		// For local Postgres it's fine to disable SSL; managed Postgres should use DB_URL.
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)
	}

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

	// Apply migrations (admin columns, otp_codes, email_queue, etc.)
	if err = ApplyMigrations(); err != nil {
		// Migrations are idempotent, but we still want visibility.
		log.Printf("Migration note: %v", err)
	}

	return nil
}

// InitSchema creates the database schema
func InitSchema() error {
	_, err := DB.Exec(schemaSQL)
	if err != nil {
		// Ignore errors if tables already exist
		fmt.Printf("Schema initialization note: %v\n", err)
	}

	return nil
}

// ApplyMigrations applies database/migrations.sql.
// It's safe to run multiple times because it uses IF NOT EXISTS and ALTER ... IF NOT EXISTS.
func ApplyMigrations() error {
	_, err := DB.Exec(migrationsSQL)
	if err != nil {
		return err
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

