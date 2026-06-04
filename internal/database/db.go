package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Connect opens a SQLite database at the given file path.
// It creates the file if it does not exist.
// Returns the *sql.DB instance used by every other package.
func Connect(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Verify the connection is actually alive
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Enable foreign key enforcement — SQLite disables this by default
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	// WAL mode gives better concurrent read performance
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	return db, nil
}