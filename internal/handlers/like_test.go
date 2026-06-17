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

func TestUpsertPostLike_AddAndCount(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()

	userID := seedLikeTestUser(t, db, "ronnie", "ronnie@test.com")
	postID, err := insertPost(db, &models.Post{UserID: userID, Title: "T", Body: "B"})
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}

	if err := upsertPostLike(db, postID, userID, 1); err != nil {
		t.Fatalf("upsertPostLike: %v", err)
	}

	likes, dislikes, err := countPostLikes(db, postID)
	if err != nil {
		t.Fatalf("countPostLikes: %v", err)
	}
	if likes != 1 || dislikes != 0 {
		t.Errorf("got likes=%d dislikes=%d, want 1/0", likes, dislikes)
	}
}

func TestUpsertPostLike_Toggle(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()

	userID := seedLikeTestUser(t, db, "ronnie", "ronnie@test.com")
	postID, _ := insertPost(db, &models.Post{UserID: userID, Title: "T", Body: "B"})

	
	if err := upsertPostLike(db, postID, userID, 1); err != nil {
		t.Fatalf("first like: %v", err)
	}
	if err := upsertPostLike(db, postID, userID, 1); err != nil {
		t.Fatalf("second like: %v", err)
	}

	likes, dislikes, err := countPostLikes(db, postID)
	if err != nil {
		t.Fatalf("countPostLikes: %v", err)
	}
	if likes != 0 || dislikes != 0 {
		t.Errorf("expected toggled-off vote, got likes=%d dislikes=%d", likes, dislikes)
	}
}

func TestUpsertPostLike_SwitchVote(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()

	userID := seedLikeTestUser(t, db, "ronnie", "ronnie@test.com")
	postID, _ := insertPost(db, &models.Post{UserID: userID, Title: "T", Body: "B"})

	if err := upsertPostLike(db, postID, userID, 1); err != nil {
		t.Fatalf("like: %v", err)
	}
	if err := upsertPostLike(db, postID, userID, -1); err != nil {
		t.Fatalf("dislike: %v", err)
	}

	likes, dislikes, err := countPostLikes(db, postID)
	if err != nil {
		t.Fatalf("countPostLikes: %v", err)
	}
	if likes != 0 || dislikes != 1 {
		t.Errorf("expected switched vote, got likes=%d dislikes=%d", likes, dislikes)
	}
}
