package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3" // adjust if the project uses a different sqlite driver

	"forum/internal/models"
)

// ── test helpers ─────────────────────────────────────────────────────────

// setupTestDB spins up a fresh in-memory SQLite DB with the minimal schema
// needed by posts.go / the post queries. Closed automatically when the
// test ends.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE
	);
	CREATE TABLE categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);
	CREATE TABLE posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);
	CREATE TABLE post_categories (
		post_id INTEGER NOT NULL,
		category_id INTEGER NOT NULL,
		PRIMARY KEY (post_id, category_id)
	);
	CREATE TABLE comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func seedUser(t *testing.T, db *sql.DB, username string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (username) VALUES (?)`, username)
	if err != nil {
		t.Fatalf("seed user %q: %v", username, err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedCategory(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO categories (name) VALUES (?)`, name)
	if err != nil {
		t.Fatalf("seed category %q: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// findProjectRoot walks up from the current working directory looking for
// web/templates/layout.html, so handler tests that render templates can run
// no matter which package directory `go test` cwd's into.
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "web", "templates", "layout.html")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// chdirProjectRoot moves the process cwd to the project root for the
// duration of the test and restores it afterward. Skips the test if the
// templates can't be located at all.
func chdirProjectRoot(t *testing.T) {
	t.Helper()
	root := findProjectRoot()
	if root == "" {
		t.Skip("could not locate web/templates from test working directory; skipping template-rendering test")
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir to project root: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

// ── insertPost / getPostByID ────────────────────────────────────────────

func TestInsertPost_AndGetPostByID(t *testing.T) {
	db := setupTestDB(t)

	uid := seedUser(t, db, "alvin")
	seedCategory(t, db, "go")
	seedCategory(t, db, "web")

	id, err := insertPost(db, &models.Post{
		UserID:     uid,
		Title:      "Hello Forum",
		Body:       "First post body",
		Categories: []string{"go", "web"},
	})
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero post id")
	}

	got, err := getPostByID(db, id)
	if err != nil {
		t.Fatalf("getPostByID: %v", err)
	}
	if got.Title != "Hello Forum" {
		t.Errorf("Title = %q, want %q", got.Title, "Hello Forum")
	}
	if got.Body != "First post body" {
		t.Errorf("Body = %q, want %q", got.Body, "First post body")
	}
	if got.Username != "alvin" {
		t.Errorf("Username = %q, want %q", got.Username, "alvin")
	}
	if len(got.Categories) != 2 {
		t.Fatalf("Categories = %v, want 2 entries", got.Categories)
	}
}

func TestInsertPost_UnknownCategory(t *testing.T) {
	db := setupTestDB(t)
	uid := seedUser(t, db, "alvin")

	_, err := insertPost(db, &models.Post{
		UserID:     uid,
		Title:      "Bad post",
		Body:       "body",
		Categories: []string{"does-not-exist"},
	})
	if err == nil {
		t.Fatal("expected an error for an unknown category, got nil")
	}
	if !strings.Contains(err.Error(), "unknown category") {
		t.Errorf("error = %v, want it to mention 'unknown category'", err)
	}
}

func TestGetPostByID_NotFound(t *testing.T) {
	db := setupTestDB(t)

	_, err := getPostByID(db, 999)
	if err != sql.ErrNoRows {
		t.Fatalf("err = %v, want sql.ErrNoRows", err)
	}
}

// ── getAllPosts ──────────────────────────────────────────────────────────

func TestGetAllPosts_OrderedNewestFirst(t *testing.T) {
	db := setupTestDB(t)
	uid := seedUser(t, db, "alvin")

	// Insert directly with explicit timestamps so ordering is deterministic
	// (insertPost always stamps datetime('now'), which can collide within a
	// single fast test run).
	rows := []struct {
		title     string
		createdAt string
	}{
		{"oldest", "2024-01-01 10:00:00"},
		{"middle", "2024-01-02 10:00:00"},
		{"newest", "2024-01-03 10:00:00"},
	}
	for _, r := range rows {
		if _, err := db.Exec(
			`INSERT INTO posts (user_id, title, content, created_at) VALUES (?, ?, ?, ?)`,
			uid, r.title, "body", r.createdAt,
		); err != nil {
			t.Fatalf("seed post %q: %v", r.title, err)
		}
	}

	posts, err := getAllPosts(db)
	if err != nil {
		t.Fatalf("getAllPosts: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("got %d posts, want 3", len(posts))
	}
	want := []string{"newest", "middle", "oldest"}
	for i, w := range want {
		if posts[i].Title != w {
			t.Errorf("posts[%d].Title = %q, want %q", i, posts[i].Title, w)
		}
	}
}

func TestGetAllPosts_Empty(t *testing.T) {
	db := setupTestDB(t)

	posts, err := getAllPosts(db)
	if err != nil {
		t.Fatalf("getAllPosts: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("got %d posts, want 0", len(posts))
	}
}

// ── getCategoriesForPost ─────────────────────────────────────────────────

func TestGetCategoriesForPost(t *testing.T) {
	db := setupTestDB(t)
	uid := seedUser(t, db, "alvin")
	seedCategory(t, db, "backend")
	seedCategory(t, db, "lightning")

	id, err := insertPost(db, &models.Post{
		UserID:     uid,
		Title:      "Bitcoin explorer notes",
		Body:       "body",
		Categories: []string{"backend", "lightning"},
	})
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}

	cats, err := getCategoriesForPost(db, id)
	if err != nil {
		t.Fatalf("getCategoriesForPost: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("got %d categories, want 2 (%v)", len(cats), cats)
	}
}

func TestGetCategoriesForPost_None(t *testing.T) {
	db := setupTestDB(t)
	uid := seedUser(t, db, "alvin")

	id, err := insertPost(db, &models.Post{
		UserID: uid,
		Title:  "No categories",
		Body:   "body",
	})
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}

	cats, err := getCategoriesForPost(db, id)
	if err != nil {
		t.Fatalf("getCategoriesForPost: %v", err)
	}
	if len(cats) != 0 {
		t.Fatalf("got %d categories, want 0", len(cats))
	}
}

// ── insertComment / getCommentsByPostID ──────────────────────────────────

func TestInsertComment_AndGetCommentsByPostID(t *testing.T) {
	db := setupTestDB(t)
	uid := seedUser(t, db, "alvin")

	postID, err := insertPost(db, &models.Post{UserID: uid, Title: "t", Body: "b"})
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}

	for _, body := range []string{"first comment", "second comment"} {
		if _, err := insertComment(db, &models.Comment{
			PostID: postID,
			UserID: uid,
			Body:   body,
		}); err != nil {
			t.Fatalf("insertComment(%q): %v", body, err)
		}
	}

	comments, err := getCommentsByPostID(db, postID)
	if err != nil {
		t.Fatalf("getCommentsByPostID: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("got %d comments, want 2", len(comments))
	}
	if comments[0].Body != "first comment" || comments[1].Body != "second comment" {
		t.Errorf("comments in unexpected order: %q, %q", comments[0].Body, comments[1].Body)
	}
	if comments[0].Username != "alvin" {
		t.Errorf("Username = %q, want %q", comments[0].Username, "alvin")
	}
}

func TestGetCommentsByPostID_Empty(t *testing.T) {
	db := setupTestDB(t)
	uid := seedUser(t, db, "alvin")

	postID, err := insertPost(db, &models.Post{UserID: uid, Title: "t", Body: "b"})
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}

	comments, err := getCommentsByPostID(db, postID)
	if err != nil {
		t.Fatalf("getCommentsByPostID: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("got %d comments, want 0", len(comments))
	}
}

// ── HTTP handlers ─────────────────────────────────────────────────────────
//
// The two "unauthenticated" tests assume auth.GetSessionUser returns an
// error (without touching the DB) when the request has no session cookie —
// the standard pattern. If Walter's implementation differs, these may need
// a stub cookie instead.

func TestNewPostGET_Unauthenticated_RedirectsToLogin(t *testing.T) {
	db := setupTestDB(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/posts/new", nil)
	rr := httptest.NewRecorder()

	h.NewPostGET(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if loc := rr.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want %q", loc, "/login")
	}
}

func TestNewPostPOST_Unauthenticated_ReturnsUnauthorized(t *testing.T) {
	db := setupTestDB(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/posts/new", nil)
	rr := httptest.NewRecorder()

	h.NewPostPOST(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestViewPost_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/posts/not-a-number", nil)
	rr := httptest.NewRecorder()

	h.ViewPost(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestViewPost_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/posts/999", nil)
	rr := httptest.NewRecorder()

	h.ViewPost(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// The two tests below actually execute the HTML templates, so they need
// web/templates to be reachable from cwd. They skip themselves if the repo
// root can't be located (e.g. running this file outside the real project
// tree).

func TestListPosts_Renders(t *testing.T) {
	chdirProjectRoot(t)
	db := setupTestDB(t)
	uid := seedUser(t, db, "alvin")
	if _, err := insertPost(db, &models.Post{UserID: uid, Title: "Hello", Body: "World"}); err != nil {
		t.Fatalf("insertPost: %v", err)
	}
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ListPosts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestViewPost_Found(t *testing.T) {
	chdirProjectRoot(t)
	db := setupTestDB(t)
	uid := seedUser(t, db, "alvin")
	postID, err := insertPost(db, &models.Post{UserID: uid, Title: "Hello", Body: "World"})
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}
	h := NewPostHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/posts/"+strconv.FormatInt(postID, 10), nil)
	rr := httptest.NewRecorder()

	h.ViewPost(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}