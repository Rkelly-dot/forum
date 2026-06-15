package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite DB with the users and sessions tables
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			email      TEXT    NOT NULL UNIQUE,
			username   TEXT    NOT NULL UNIQUE,
			password   TEXT    NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			token      TEXT    NOT NULL UNIQUE,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create sessions table: %v", err)
	}

	return db
}

func TestRegisterPOST_Success(t *testing.T) {
	db := setupTestDB(t)
	h := NewAuthHandler(db)

	form := url.Values{}
	form.Set("username", "walter")
	form.Set("email", "walter@test.com")
	form.Set("password", "secret123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	h.RegisterPOST(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "walter@test.com").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 user in db, got %d", count)
	}
}

func TestRegisterPOST_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	h := NewAuthHandler(db)

	form := url.Values{}
	form.Set("username", "walter")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	h.RegisterPOST(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestRegisterPOST_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	h := NewAuthHandler(db)

	db.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		"walter", "walter@test.com", "hashedpw",
	)

	form := url.Values{}
	form.Set("username", "walter2")
	form.Set("email", "walter@test.com")
	form.Set("password", "anotherpass")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	h.RegisterPOST(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestLoginPOST_Success(t *testing.T) {
	db := setupTestDB(t)
	h := NewAuthHandler(db)

	form := url.Values{}
	form.Set("username", "walter")
	form.Set("email", "walter@test.com")
	form.Set("password", "secret123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.RegisterPOST(httptest.NewRecorder(), req)

	loginForm := url.Values{}
	loginForm.Set("email", "walter@test.com")
	loginForm.Set("password", "secret123")

	req2 := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	h.LoginPOST(rr, req2)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}

	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session_token" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected session_token cookie to be set")
	}

	// confirm a session row exists in the db
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 session row, got %d", count)
	}
}

func TestLoginPOST_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	h := NewAuthHandler(db)

	form := url.Values{}
	form.Set("username", "walter")
	form.Set("email", "walter@test.com")
	form.Set("password", "secret123")
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.RegisterPOST(httptest.NewRecorder(), req)

	loginForm := url.Values{}
	loginForm.Set("email", "walter@test.com")
	loginForm.Set("password", "wrongpassword")

	req2 := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	h.LoginPOST(rr, req2)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestLoginPOST_NonexistentUser(t *testing.T) {
	db := setupTestDB(t)
	h := NewAuthHandler(db)

	loginForm := url.Values{}
	loginForm.Set("email", "nobody@test.com")
	loginForm.Set("password", "whatever")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	h.LoginPOST(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestLogout(t *testing.T) {
	db := setupTestDB(t)
	h := NewAuthHandler(db)

	// register and log in to get a real session
	form := url.Values{}
	form.Set("username", "walter")
	form.Set("email", "walter@test.com")
	form.Set("password", "secret123")
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.RegisterPOST(httptest.NewRecorder(), req)

	loginForm := url.Values{}
	loginForm.Set("email", "walter@test.com")
	loginForm.Set("password", "secret123")
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRR := httptest.NewRecorder()
	h.LoginPOST(loginRR, loginReq)

	var token string
	for _, c := range loginRR.Result().Cookies() {
		if c.Name == "session_token" {
			token = c.Value
		}
	}

	// now log out
	req2 := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req2.AddCookie(&http.Cookie{Name: "session_token", Value: token})

	rr := httptest.NewRecorder()
	h.Logout(rr, req2)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&count)
	if count != 0 {
		t.Errorf("expected session to be deleted, got %d rows", count)
	}
}
