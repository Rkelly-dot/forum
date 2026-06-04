package database

import (
	"database/sql"
	"fmt"
)

// RunMigrations creates all tables if they do not already exist.
// Safe to call on every startup — uses CREATE TABLE IF NOT EXISTS throughout.
func RunMigrations(db *sql.DB) error {
	statements := []struct {
		name string
		sql  string
	}{
		{
			name: "users",
			sql: `
			CREATE TABLE IF NOT EXISTS users (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				email      TEXT    NOT NULL UNIQUE,
				username   TEXT    NOT NULL UNIQUE,
				password   TEXT    NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "sessions",
			sql: `
			CREATE TABLE IF NOT EXISTS sessions (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				token      TEXT    NOT NULL UNIQUE,
				user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				expires_at DATETIME NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "categories",
			sql: `
			CREATE TABLE IF NOT EXISTS categories (
				id   INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT    NOT NULL UNIQUE
			)`,
		},
		{
			name: "posts",
			sql: `
			CREATE TABLE IF NOT EXISTS posts (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				title      TEXT    NOT NULL,
				content    TEXT    NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "post_categories",
			sql: `
			CREATE TABLE IF NOT EXISTS post_categories (
				post_id     INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
				category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
				PRIMARY KEY (post_id, category_id)
			)`,
		},
		{
			name: "comments",
			sql: `
			CREATE TABLE IF NOT EXISTS comments (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
				user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				content    TEXT    NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
		},
		{
			name: "likes",
			sql: `
			CREATE TABLE IF NOT EXISTS likes (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				post_id    INTEGER REFERENCES posts(id)    ON DELETE CASCADE,
				comment_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
				value      INTEGER NOT NULL CHECK(value IN (1, -1)),
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				-- A user can only vote once per post or comment
				UNIQUE (user_id, post_id),
				UNIQUE (user_id, comment_id),
				-- Every like must target exactly one of post or comment, not both
				CHECK (
					(post_id IS NOT NULL AND comment_id IS NULL) OR
					(post_id IS NULL AND comment_id IS NOT NULL)
				)
			)`,
		},
	}

	for _, s := range statements {
		if _, err := db.Exec(s.sql); err != nil {
			return fmt.Errorf("migration %q failed: %w", s.name, err)
		}
	}

	// Seed default categories if table is empty
	if err := seedCategories(db); err != nil {
		return fmt.Errorf("seed categories: %w", err)
	}

	return nil
}

// seedCategories inserts a default set of categories on first run.
// Uses INSERT OR IGNORE so it's safe to run on every startup.
func seedCategories(db *sql.DB) error {
	defaults := []string{"General", "Technology", "Gaming", "Science", "Sports", "Music", "Other"}
	for _, name := range defaults {
		if _, err := db.Exec(`INSERT OR IGNORE INTO categories (name) VALUES (?)`, name); err != nil {
			return fmt.Errorf("seed category %q: %w", name, err)
		}
	}
	return nil
}