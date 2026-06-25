package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// ── test DB setup ──────────────────────────────────────────────────────────

func newLikeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		email      TEXT    NOT NULL UNIQUE,
		username   TEXT    NOT NULL UNIQUE,
		password   TEXT    NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS categories (
		id   INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT    NOT NULL UNIQUE
	);
	CREATE TABLE IF NOT EXISTS posts (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title      TEXT    NOT NULL,
		content    TEXT    NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS post_categories (
		post_id     INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
		PRIMARY KEY (post_id, category_id)
	);
	CREATE TABLE IF NOT EXISTS comments (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		content    TEXT    NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS likes (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		post_id    INTEGER REFERENCES posts(id)    ON DELETE CASCADE,
		comment_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
		value      INTEGER NOT NULL CHECK(value IN (1, -1)),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (user_id, post_id),
		UNIQUE (user_id, comment_id)
	);
	CREATE TABLE IF NOT EXISTS sessions (
		id         TEXT PRIMARY KEY,
		token      TEXT    NOT NULL UNIQUE,
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return db
}

// seed helpers — prefixed lt_ to avoid conflicts with post_test.go helpers

func ltSeedUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO users (username, email, password, created_at)
		 VALUES ('alice', 'alice@test.com', 'hashed', datetime('now'))`,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func ltSeedPost(t *testing.T, db *sql.DB, userID int64) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO posts (user_id, title, content, created_at)
		 VALUES (?, 'Test Post', 'body', datetime('now'))`,
		userID,
	)
	if err != nil {
		t.Fatalf("seed post: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func ltSeedComment(t *testing.T, db *sql.DB, postID, userID int64) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO comments (post_id, user_id, content, created_at)
		 VALUES (?, ?, 'A comment', datetime('now'))`,
		postID, userID,
	)
	if err != nil {
		t.Fatalf("seed comment: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func ltSeedSession(t *testing.T, db *sql.DB, userID int64) string {
	t.Helper()
	sid := "like-test-session-abc"
	_, err := db.Exec(
		`INSERT INTO sessions (id, token, user_id, expires_at)
		 VALUES (?, ?, ?, datetime('now', '+24 hours'))`,
		sid, sid, userID,
	)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return sid
}

func id64(n int64) string { return strconv.FormatInt(n, 10) }

// ── acceptsJSON ────────────────────────────────────────────────────────────

func TestAcceptsJSON(t *testing.T) {
	cases := []struct {
		header string
		want   bool
	}{
		{"application/json", true},
		{"application/json, */*", true},
		{"*/*", true},
		{"text/html", false},
		{"", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		if c.header != "" {
			r.Header.Set("Accept", c.header)
		}
		got := acceptsJSON(r)
		if got != c.want {
			t.Errorf("acceptsJSON(header=%q) = %v, want %v", c.header, got, c.want)
		}
	}
}

// ── upsertPostLike ─────────────────────────────────────────────────────────

func TestUpsertPostLike_Insert(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)

	if err := upsertPostLike(db, postID, userID, 1); err != nil {
		t.Fatalf("upsertPostLike: %v", err)
	}

	var value int
	if err := db.QueryRow(
		`SELECT value FROM likes WHERE post_id = ? AND user_id = ? AND comment_id IS NULL`,
		postID, userID,
	).Scan(&value); err != nil {
		t.Fatalf("row not found after insert: %v", err)
	}
	if value != 1 {
		t.Errorf("value = %d, want 1", value)
	}
}

func TestUpsertPostLike_Toggle(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)

	upsertPostLike(db, postID, userID, 1)
	// voting same value again removes the like
	if err := upsertPostLike(db, postID, userID, 1); err != nil {
		t.Fatalf("upsertPostLike toggle: %v", err)
	}

	var count int
	db.QueryRow(
		`SELECT COUNT(*) FROM likes WHERE post_id = ? AND user_id = ? AND comment_id IS NULL`,
		postID, userID,
	).Scan(&count)
	if count != 0 {
		t.Errorf("expected row deleted after toggle, got count=%d", count)
	}
}

func TestUpsertPostLike_SwitchVote(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)

	upsertPostLike(db, postID, userID, 1)
	if err := upsertPostLike(db, postID, userID, -1); err != nil {
		t.Fatalf("upsertPostLike switch: %v", err)
	}

	var value int
	db.QueryRow(
		`SELECT value FROM likes WHERE post_id = ? AND user_id = ? AND comment_id IS NULL`,
		postID, userID,
	).Scan(&value)
	if value != -1 {
		t.Errorf("value = %d, want -1 after switch", value)
	}
}

// ── upsertCommentLike ──────────────────────────────────────────────────────

func TestUpsertCommentLike_Insert(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)
	commentID := ltSeedComment(t, db, postID, userID)

	if err := upsertCommentLike(db, commentID, userID, -1); err != nil {
		t.Fatalf("upsertCommentLike: %v", err)
	}

	var value int
	if err := db.QueryRow(
		`SELECT value FROM likes WHERE comment_id = ? AND user_id = ? AND post_id IS NULL`,
		commentID, userID,
	).Scan(&value); err != nil {
		t.Fatalf("row not found: %v", err)
	}
	if value != -1 {
		t.Errorf("value = %d, want -1", value)
	}
}

func TestUpsertCommentLike_Toggle(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)
	commentID := ltSeedComment(t, db, postID, userID)

	upsertCommentLike(db, commentID, userID, 1)
	upsertCommentLike(db, commentID, userID, 1) // same → delete

	var count int
	db.QueryRow(
		`SELECT COUNT(*) FROM likes WHERE comment_id = ? AND user_id = ? AND post_id IS NULL`,
		commentID, userID,
	).Scan(&count)
	if count != 0 {
		t.Errorf("expected row deleted after toggle, got count=%d", count)
	}
}

func TestUpsertCommentLike_SwitchVote(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)
	commentID := ltSeedComment(t, db, postID, userID)

	upsertCommentLike(db, commentID, userID, 1)
	upsertCommentLike(db, commentID, userID, -1)

	var value int
	db.QueryRow(
		`SELECT value FROM likes WHERE comment_id = ? AND user_id = ? AND post_id IS NULL`,
		commentID, userID,
	).Scan(&value)
	if value != -1 {
		t.Errorf("value = %d, want -1 after switch", value)
	}
}

// ── countPostLikes ─────────────────────────────────────────────────────────

func TestCountPostLikes_Zero(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)

	likes, dislikes, err := countPostLikes(db, postID)
	if err != nil {
		t.Fatalf("countPostLikes: %v", err)
	}
	if likes != 0 || dislikes != 0 {
		t.Errorf("expected 0/0, got %d/%d", likes, dislikes)
	}
}

func TestCountPostLikes_Mixed(t *testing.T) {
	db := newLikeTestDB(t)
	user1 := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, user1)

	// two more users to cast independent votes
	u2res, _ := db.Exec(`INSERT INTO users (username,email,password,created_at) VALUES ('bob','bob@t.com','h',datetime('now'))`)
	u2, _ := u2res.LastInsertId()
	u3res, _ := db.Exec(`INSERT INTO users (username,email,password,created_at) VALUES ('carol','carol@t.com','h',datetime('now'))`)
	u3, _ := u3res.LastInsertId()

	upsertPostLike(db, postID, user1, 1)
	upsertPostLike(db, postID, u2, 1)
	upsertPostLike(db, postID, u3, -1)

	likes, dislikes, err := countPostLikes(db, postID)
	if err != nil {
		t.Fatalf("countPostLikes: %v", err)
	}
	if likes != 2 || dislikes != 1 {
		t.Errorf("got likes=%d dislikes=%d, want 2/1", likes, dislikes)
	}
}

// ── countCommentLikes ──────────────────────────────────────────────────────

func TestCountCommentLikes_Zero(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)
	commentID := ltSeedComment(t, db, postID, userID)

	likes, dislikes, err := countCommentLikes(db, commentID)
	if err != nil {
		t.Fatalf("countCommentLikes: %v", err)
	}
	if likes != 0 || dislikes != 0 {
		t.Errorf("expected 0/0, got %d/%d", likes, dislikes)
	}
}

func TestCountCommentLikes_AfterVote(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)
	commentID := ltSeedComment(t, db, postID, userID)

	upsertCommentLike(db, commentID, userID, -1)

	likes, dislikes, err := countCommentLikes(db, commentID)
	if err != nil {
		t.Fatalf("countCommentLikes: %v", err)
	}
	if likes != 0 || dislikes != 1 {
		t.Errorf("got likes=%d dislikes=%d, want 0/1", likes, dislikes)
	}
}

// ── getUserPostVote ────────────────────────────────────────────────────────

func TestGetUserPostVote_Exists(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)
	upsertPostLike(db, postID, userID, 1)

	v, err := getUserPostVote(db, postID, userID)
	if err != nil || v != 1 {
		t.Errorf("got v=%d err=%v, want v=1 err=nil", v, err)
	}
}

func TestGetUserPostVote_NotFound(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)

	v, err := getUserPostVote(db, postID, userID)
	if err != nil || v != 0 {
		t.Errorf("got v=%d err=%v, want v=0 err=nil", v, err)
	}
}

// ── getUserCommentVote ─────────────────────────────────────────────────────

func TestGetUserCommentVote_Exists(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)
	commentID := ltSeedComment(t, db, postID, userID)
	upsertCommentLike(db, commentID, userID, -1)

	v, err := getUserCommentVote(db, commentID, userID)
	if err != nil || v != -1 {
		t.Errorf("got v=%d err=%v, want v=-1 err=nil", v, err)
	}
}

func TestGetUserCommentVote_NotFound(t *testing.T) {
	db := newLikeTestDB(t)
	userID := ltSeedUser(t, db)
	postID := ltSeedPost(t, db, userID)
	commentID := ltSeedComment(t, db, postID, userID)

	v, err := getUserCommentVote(db, commentID, userID)
	if err != nil || v != 0 {
		t.Errorf("got v=%d err=%v, want v=0 err=nil", v, err)
	}
}

// ── LikeHandler HTTP ───────────────────────────────────────────────────────

func TestLikeHandler_WrongMethod(t *testing.T) {
	db := newLikeTestDB(t)
	h := NewLikeHandler(db)

	r := httptest.NewRequest(http.MethodGet, "/like", nil)
	w := httptest.NewRecorder()
	h.Like(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

func TestLikeHandler_GuestUnauthorized(t *testing.T) {
	db := newLikeTestDB(t)
	h := NewLikeHandler(db)

	form := url.Values{"value": {"1"}, "post_id": {"1"}}
	r := httptest.NewRequest(http.MethodPost, "/like", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	h.Like(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", w.Code)
	}
}


func TestWriteJSONCounts_Shape(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONCounts(w, 5, 2)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got map[string]int
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["likes"] != 5 || got["dislikes"] != 2 {
		t.Errorf("payload = %v, want {likes:5 dislikes:2}", got)
	}
}