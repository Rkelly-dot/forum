package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"forum/internal/models"
)

func setupLikeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		username   TEXT NOT NULL UNIQUE,
		email      TEXT NOT NULL UNIQUE,
		password   TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS posts (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id    INTEGER NOT NULL REFERENCES users(id),
		title      TEXT NOT NULL,
		body       TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS post_categories (
		post_id  INTEGER NOT NULL REFERENCES posts(id),
		category TEXT NOT NULL,
		PRIMARY KEY (post_id, category)
	);
	CREATE TABLE IF NOT EXISTS comments (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id    INTEGER NOT NULL REFERENCES posts(id),
		user_id    INTEGER NOT NULL REFERENCES users(id),
		body       TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS likes (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id    INTEGER REFERENCES posts(id),
		comment_id INTEGER REFERENCES comments(id),
		user_id    INTEGER NOT NULL REFERENCES users(id),
		value      INTEGER NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS sessions (
		id         TEXT PRIMARY KEY,
		user_id    INTEGER NOT NULL REFERENCES users(id),
		expires_at TEXT NOT NULL
	);`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return db
}

func seedLikeTestUser(t *testing.T, db *sql.DB, username, email string) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO users (username, email, password, created_at) VALUES (?, ?, 'hashed', datetime('now'))`,
		username, email,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}


