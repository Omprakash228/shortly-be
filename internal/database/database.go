package database

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// NewConnection creates a new database connection
func NewConnection(databaseURL string) (*sql.DB, error) {
	// Open connection to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("✅ Successfully connected to database!")
	return db, nil
}

// RunMigrations runs database migrations using goose
func RunMigrations(db *sql.DB) error {
	// Get the migrations directory path
	migrationsDir := "migrations"
	
	// Set the dialect to postgres
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Run migrations
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println("✅ Database migrations completed!")
	return nil
}

