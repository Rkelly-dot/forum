package handlers
 
import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
 
	_ "github.com/mattn/go-sqlite3"
)
 
// ─── Test DB Setup ────────────────────────────────────────────────────────────
 
func setupLikeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
 
	schema := `
	CREATE TABLE users (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		username   TEXT NOT NULL UNIQUE,
		email      TEXT NOT NULL UNIQUE,
		password   TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE TABLE sessions (
		token      TEXT PRIMARY KEY,
		user_id    INTEGER NOT NULL REFERENCES users(id),
		expires_at TEXT NOT NULL
	);
	CREATE TABLE posts (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id    INTEGER NOT NULL REFERENCES users(id),
		title      TEXT NOT NULL,
		content    TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE TABLE comments (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id    INTEGER NOT NULL REFERENCES posts(id),
		user_id    INTEGER NOT NULL REFERENCES users(id),
		content    TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE TABLE likes (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id    INTEGER REFERENCES posts(id),
		comment_id INTEGER REFERENCES comments(id),
		user_id    INTEGER NOT NULL REFERENCES users(id),
		value      INTEGER NOT NULL CHECK(value IN (1,-1)),
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(post_id, user_id),
		UNIQUE(comment_id, user_id)
	);`
 
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}
 
// seed inserts a user, a session cookie, a post and a comment; returns their IDs.
func seed(t *testing.T, db *sql.DB) (userID, postID, commentID int64, sessionToken string) {
	t.Helper()
 
	res, err := db.Exec(
		`INSERT INTO users (username, email, password) VALUES ('alice','alice@example.com','hashed')`,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID, _ = res.LastInsertId()
 
	sessionToken = "test-session-token"
	_, err = db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, datetime('now','+1 hour'))`,
		sessionToken, userID,
	)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}
 
	res, err = db.Exec(
		`INSERT INTO posts (user_id, title, content) VALUES (?, 'Hello', 'World')`, userID,
	)
	if err != nil {
		t.Fatalf("insert post: %v", err)
	}
	postID, _ = res.LastInsertId()
 
	res, err = db.Exec(
		`INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, 'Nice post')`, postID, userID,
	)
	if err != nil {
		t.Fatalf("insert comment: %v", err)
	}
	commentID, _ = res.LastInsertId()
 
	return
}
 
 
 
// ─── upsertPostLike / countPostLikes ─────────────────────────────────────────
 
func TestUpsertPostLike_Insert(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()
	_, postID, _, _ := seed(t, db)
 
	if err := upsertPostLike(db, postID, 1, 1); err != nil {
		t.Fatalf("upsert: %v", err)
	}
 
	likes, dislikes, err := countPostLikes(db, postID)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if likes != 1 || dislikes != 0 {
		t.Errorf("want likes=1 dislikes=0, got likes=%d dislikes=%d", likes, dislikes)
	}
}
 
func TestUpsertPostLike_Toggle(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()
	_, postID, _, _ := seed(t, db)
 
	// like then like again → should remove the like (toggle off)
	_ = upsertPostLike(db, postID, 1, 1)
	if err := upsertPostLike(db, postID, 1, 1); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
 
	likes, dislikes, _ := countPostLikes(db, postID)
	if likes != 0 || dislikes != 0 {
		t.Errorf("expected 0/0 after toggle, got %d/%d", likes, dislikes)
	}
}

func TestUpsertPostLike_Flip(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()
	_, postID, _, _ := seed(t, db)
 
	_ = upsertPostLike(db, postID, 1, 1)   // like
	_ = upsertPostLike(db, postID, 1, -1)  // flip to dislike
 
	likes, dislikes, _ := countPostLikes(db, postID)
	if likes != 0 || dislikes != 1 {
		t.Errorf("expected 0/1 after flip, got %d/%d", likes, dislikes)
	}
}
 
// ─── upsertCommentLike / countCommentLikes ───────────────────────────────────
 
func TestUpsertCommentLike_Insert(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()
	_, _, commentID, _ := seed(t, db)
 
	if err := upsertCommentLike(db, commentID, 1, -1); err != nil {
		t.Fatalf("upsert: %v", err)
	}
 
	likes, dislikes, _ := countCommentLikes(db, commentID)
	if likes != 0 || dislikes != 1 {
		t.Errorf("want 0/1, got %d/%d", likes, dislikes)
	}
}


