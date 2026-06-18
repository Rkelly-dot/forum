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

func TestUpsertCommentLike_AddAndCount(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()

	userID := seedLikeTestUser(t, db, "ronnie", "ronnie@test.com")
	postID, _ := insertPost(db, &models.Post{UserID: userID, Title: "T", Body: "B"})
	commentID, err := insertComment(db, &models.Comment{PostID: postID, UserID: userID, Body: "nice"})
	if err != nil {
		t.Fatalf("insertComment: %v", err)
	}

	if err := upsertCommentLike(db, commentID, userID, -1); err != nil {
		t.Fatalf("upsertCommentLike: %v", err)
	}

	likes, dislikes, err := countCommentLikes(db, commentID)
	if err != nil {
		t.Fatalf("countCommentLikes: %v", err)
	}
	if likes != 0 || dislikes != 1 {
		t.Errorf("got likes=%d dislikes=%d, want 0/1", likes, dislikes)
	}
}

func TestGetPostsLikedByUser(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()

	userID := seedLikeTestUser(t, db, "ronnie", "ronnie@test.com")
	postA, _ := insertPost(db, &models.Post{UserID: userID, Title: "A", Body: "B"})
	_, _ = insertPost(db, &models.Post{UserID: userID, Title: "Unliked", Body: "B"})

	if err := upsertPostLike(db, postA, userID, 1); err != nil {
		t.Fatalf("upsertPostLike: %v", err)
	}

	liked, err := getPostsLikedByUser(db, userID)
	if err != nil {
		t.Fatalf("getPostsLikedByUser: %v", err)
	}
	if len(liked) != 1 {
		t.Errorf("got %d liked posts, want 1", len(liked))
	}
}

func TestGetPostsByCategory(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()

	userID := seedLikeTestUser(t, db, "ronnie", "ronnie@test.com")
	_, _ = insertPost(db, &models.Post{UserID: userID, Title: "Tech post", Body: "B", Categories: []string{"tech"}})
	_, _ = insertPost(db, &models.Post{UserID: userID, Title: "Other post", Body: "B", Categories: []string{"general"}})

	posts, err := getPostsByCategory(db, "tech")
	if err != nil {
		t.Fatalf("getPostsByCategory: %v", err)
	}
	if len(posts) != 1 {
		t.Errorf("got %d posts, want 1", len(posts))
	}
}

func TestLike_GuestUnauthorized(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()

	userID := seedLikeTestUser(t, db, "owner", "owner@test.com")
	postID, _ := insertPost(db, &models.Post{UserID: userID, Title: "T", Body: "B"})

	handler := NewLikeHandler(db)

	form := url.Values{}
	form.Set("post_id", strconv.FormatInt(postID, 10))
	form.Set("value", "1")

	req := httptest.NewRequest(http.MethodPost, "/like", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.Like(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestLike_InvalidValue(t *testing.T) {
	db := setupLikeTestDB(t)
	defer db.Close()

	handler := NewLikeHandler(db)

	form := url.Values{}
	form.Set("post_id", "1")
	form.Set("value", "5")

	req := httptest.NewRequest(http.MethodPost, "/like", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.Like(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for guest request, got %d", rr.Code)
	}
}
