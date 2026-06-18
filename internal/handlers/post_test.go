package handlers
 
import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
 
	_ "github.com/mattn/go-sqlite3"
	"forum/internal/models"
)
 

 
func setupTestDB(t *testing.T) *sql.DB {
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
 
func seedUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO users (username, email, password, created_at) VALUES ('alvin', 'alvin@test.com', 'hashed', datetime('now'))`,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}
 
func seedSession(t *testing.T, db *sql.DB, userID int64) string {
	t.Helper()
	sessionID := "test-session-id-abc123"
	_, err := db.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, datetime('now', '+24 hours'))`,
		sessionID, userID,
	)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return sessionID
}


// ── query-level tests ──────────────────────────────────────────────────────
 
func TestInsertPost(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	userID := seedUser(t, db)
 
	p := &models.Post{
		UserID:     userID,
		Title:      "Hello Kisumu",
		Body:       "First post body",
		Categories: []string{"tech", "general"},
	}
 
	id, err := insertPost(db, p)
	if err != nil {
		t.Fatalf("insertPost: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero post id")
	}
 
	fetched, err := getPostByID(db, id)
	if err != nil {
		t.Fatalf("getPostByID: %v", err)
	}
	if fetched.Title != p.Title {
		t.Errorf("title: got %q want %q", fetched.Title, p.Title)
	}
	if len(fetched.Categories) != 2 {
		t.Errorf("categories: got %d want 2", len(fetched.Categories))
	}
}
 
func TestGetAllPosts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	userID := seedUser(t, db)
 
	for i := 0; i < 3; i++ {
		_, err := insertPost(db, &models.Post{
			UserID: userID,
			Title:  "Post",
			Body:   "Body",
		})
		if err != nil {
			t.Fatalf("insertPost: %v", err)
		}
	}
 
	posts, err := getAllPosts(db)
	if err != nil {
		t.Fatalf("getAllPosts: %v", err)
	}
	if len(posts) != 3 {
		t.Errorf("got %d posts, want 3", len(posts))
	}
}

func TestInsertAndFetchComments(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	userID := seedUser(t, db)
 
	postID, _ := insertPost(db, &models.Post{
		UserID: userID,
		Title:  "Thread post",
		Body:   "Body",
	})
 
	for i := 0; i < 2; i++ {
		_, err := insertComment(db, &models.Comment{
			PostID: postID,
			UserID: userID,
			Body:   "A comment",
		})
		if err != nil {
			t.Fatalf("insertComment: %v", err)
		}
	}
 
	comments, err := getCommentsByPostID(db, postID)
	if err != nil {
		t.Fatalf("getCommentsByPostID: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("got %d comments, want 2", len(comments))
	}
}
 
func TestGetPostByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
 
	_, err := getPostByID(db, 9999)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

// ── handler-level tests ────────────────────────────────────────────────────
 
func TestNewPostPOST_GuestUnauthorized(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
 
	handler := NewPostHandler(db, nil)
 
	form := url.Values{}
	form.Set("title", "Guest post attempt")
	form.Set("body", "Should not be saved")
 
	req := httptest.NewRequest(http.MethodPost, "/posts/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// no session cookie set — guest request
 
	rr := httptest.NewRecorder()
	handler.NewPostPOST(rr, req)
 
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}
 
func TestCreateComment_GuestUnauthorized(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
 
	handler := NewCommentHandler(db, nil)
 
	form := url.Values{}
	form.Set("body", "Guest comment attempt")
 
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
 
	rr := httptest.NewRecorder()
	handler.CreateComment(rr, req)
 
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestNewPostGET_GuestRedirects(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
 
	handler := NewPostHandler(db, nil)
 
	req := httptest.NewRequest(http.MethodGet, "/posts/new", nil)
	rr := httptest.NewRecorder()
	handler.NewPostGET(rr, req)
 
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303 redirect to /login, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/login" {
		t.Errorf("expected redirect to /login, got %q", loc)
	}
}
 