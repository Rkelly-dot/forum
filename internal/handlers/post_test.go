package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"forum/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// ── DB setup ───────────────────────────────────────────────────────────────

func newPostTestDB(t *testing.T) *sql.DB {
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
	for _, name := range []string{"General", "Technology", "Gaming", "Science", "Sports", "Music", "Other"} {
		if _, err := db.Exec(`INSERT OR IGNORE INTO categories (name) VALUES (?)`, name); err != nil {
			t.Fatalf("seed category %q: %v", name, err)
		}
	}
	return db
}

func ptSeedUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO users (username, email, password, created_at)
		 VALUES ('alvin', 'alvin@test.com', 'hashed', datetime('now'))`,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func ptSeedSession(t *testing.T, db *sql.DB, userID int64) string {
	t.Helper()
	sid := "test-session-post-xyz"
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

// ── insertPost ─────────────────────────────────────────────────────────────

func TestInsertPost_NoCategories(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)

	id, err := insertPost(db, &models.Post{
		UserID: userID,
		Title:  "No cats",
		Body:   "Body text",
	})
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM posts WHERE id = ?`, id).Scan(&count)
	if count != 1 {
		t.Errorf("post not found in DB after insert")
	}
}

func TestInsertPost_WithCategories(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)

	id, err := insertPost(db, &models.Post{
		UserID:     userID,
		Title:      "With cats",
		Body:       "Body",
		Categories: []string{"Technology", "Gaming"},
	})
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM post_categories WHERE post_id = ?`, id).Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 post_categories rows, got %d", count)
	}
}

func TestInsertPost_UnknownCategory(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)

	_, err := insertPost(db, &models.Post{
		UserID:     userID,
		Title:      "Bad cat",
		Body:       "Body",
		Categories: []string{"DoesNotExist"},
	})
	if err == nil {
		t.Fatal("expected error for unknown category, got nil")
	}
}

// ── getPostByID ────────────────────────────────────────────────────────────

func TestGetPostByID_Found(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)

	id, _ := insertPost(db, &models.Post{
		UserID:     userID,
		Title:      "Kisumu Tech",
		Body:       "Great event",
		Categories: []string{"Technology"},
	})

	p, err := getPostByID(db, id)
	if err != nil {
		t.Fatalf("getPostByID: %v", err)
	}
	if p.Title != "Kisumu Tech" {
		t.Errorf("title: got %q, want %q", p.Title, "Kisumu Tech")
	}
	if p.Username != "alvin" {
		t.Errorf("username: got %q, want %q", p.Username, "alvin")
	}
	if len(p.Categories) != 1 || p.Categories[0] != "Technology" {
		t.Errorf("categories: got %v, want [Technology]", p.Categories)
	}
}

func TestGetPostByID_NotFound(t *testing.T) {
	db := newPostTestDB(t)

	_, err := getPostByID(db, 99999)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

// ── getAllPosts ─────────────────────────────────────────────────────────────

func TestGetAllPosts_Empty(t *testing.T) {
	db := newPostTestDB(t)

	posts, err := getAllPosts(db)
	if err != nil {
		t.Fatalf("getAllPosts: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}


func TestGetCategoriesForPost_None(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)

	id, _ := insertPost(db, &models.Post{UserID: userID, Title: "No cats", Body: "b"})
	cats, err := getCategoriesForPost(db, id)
	if err != nil {
		t.Fatalf("getCategoriesForPost: %v", err)
	}
	if len(cats) != 0 {
		t.Errorf("expected 0 categories, got %v", cats)
	}
}

func TestGetCategoriesForPost_Several(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)

	id, _ := insertPost(db, &models.Post{
		UserID:     userID,
		Title:      "Multi",
		Body:       "b",
		Categories: []string{"General", "Music", "Gaming"},
	})
	cats, err := getCategoriesForPost(db, id)
	if err != nil {
		t.Fatalf("getCategoriesForPost: %v", err)
	}
	if len(cats) != 3 {
		t.Errorf("expected 3 categories, got %v", cats)
	}
}

// ── insertComment ──────────────────────────────────────────────────────────

func TestInsertComment_Success(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)
	postID, _ := insertPost(db, &models.Post{UserID: userID, Title: "T", Body: "B"})

	id, err := insertComment(db, &models.Comment{
		PostID: postID,
		UserID: userID,
		Body:   "Nice post!",
	})
	if err != nil {
		t.Fatalf("insertComment: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero comment id")
	}
}

func TestInsertComment_InvalidPostID(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)
	db.Exec(`PRAGMA foreign_keys = ON`)

	_, err := insertComment(db, &models.Comment{
		PostID: 99999,
		UserID: userID,
		Body:   "orphan",
	})
	if err == nil {
		t.Fatal("expected FK error for non-existent post, got nil")
	}
}

// ── getCommentsByPostID ────────────────────────────────────────────────────

func TestGetCommentsByPostID_Empty(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)
	postID, _ := insertPost(db, &models.Post{UserID: userID, Title: "T", Body: "B"})

	comments, err := getCommentsByPostID(db, postID)
	if err != nil {
		t.Fatalf("getCommentsByPostID: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
}

func TestGetCommentsByPostID_Multiple(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)
	postID, _ := insertPost(db, &models.Post{UserID: userID, Title: "T", Body: "B"})

	for i := 0; i < 3; i++ {
		_, err := insertComment(db, &models.Comment{PostID: postID, UserID: userID, Body: "reply"})
		if err != nil {
			t.Fatalf("insertComment: %v", err)
		}
	}

	comments, err := getCommentsByPostID(db, postID)
	if err != nil {
		t.Fatalf("getCommentsByPostID: %v", err)
	}
	if len(comments) != 3 {
		t.Errorf("expected 3 comments, got %d", len(comments))
	}
}

func TestGetCommentsByPostID_UsernameJoined(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)
	postID, _ := insertPost(db, &models.Post{UserID: userID, Title: "T", Body: "B"})
	insertComment(db, &models.Comment{PostID: postID, UserID: userID, Body: "hi"})

	comments, err := getCommentsByPostID(db, postID)
	if err != nil {
		t.Fatalf("getCommentsByPostID: %v", err)
	}
	if comments[0].Username != "alvin" {
		t.Errorf("username join: got %q, want %q", comments[0].Username, "alvin")
	}
}

func TestGetCommentsByPostID_IsolatedFromOtherPosts(t *testing.T) {
	db := newPostTestDB(t)
	userID := ptSeedUser(t, db)

	postA, _ := insertPost(db, &models.Post{UserID: userID, Title: "A", Body: "B"})
	postB, _ := insertPost(db, &models.Post{UserID: userID, Title: "B", Body: "B"})

	insertComment(db, &models.Comment{PostID: postA, UserID: userID, Body: "on A"})
	insertComment(db, &models.Comment{PostID: postA, UserID: userID, Body: "also A"})
	insertComment(db, &models.Comment{PostID: postB, UserID: userID, Body: "on B"})

	comments, err := getCommentsByPostID(db, postA)
	if err != nil {
		t.Fatalf("getCommentsByPostID: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("expected 2 comments for postA, got %d", len(comments))
	}
}

// ── HTTP handlers (no template rendering required) ─────────────────────────

func TestNewPostGET_GuestRedirectsToLogin(t *testing.T) {
	db := newPostTestDB(t)
	h := NewPostHandler(db)

	r := httptest.NewRequest(http.MethodGet, "/posts/new", nil)
	w := httptest.NewRecorder()
	h.NewPostGET(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Errorf("expected redirect to /login, got %q", loc)
	}
}

func TestNewPostPOST_GuestReturns401(t *testing.T) {
	db := newPostTestDB(t)
	h := NewPostHandler(db)

	form := url.Values{"title": {"Sneaky"}, "content": {"body"}}
	r := httptest.NewRequest(http.MethodPost, "/posts/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	h.NewPostPOST(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestViewPost_NotFound(t *testing.T) {
	db := newPostTestDB(t)
	h := NewPostHandler(db)

	r := httptest.NewRequest(http.MethodGet, "/posts/99999", nil)
	w := httptest.NewRecorder()
	h.ViewPost(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestViewPost_BadID(t *testing.T) {
	db := newPostTestDB(t)
	h := NewPostHandler(db)

	r := httptest.NewRequest(http.MethodGet, "/posts/notanid", nil)
	w := httptest.NewRecorder()
	h.ViewPost(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}