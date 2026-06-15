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

// setupTestDB creates an in-memory SQLite DB with the users table
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email    TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
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

	// confirm user was inserted
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
	// email and password missing

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

	// insert first user directly
	db.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		"walter", "walter@test.com", "hashedpw",
	)

	form := url.Values{}
	form.Set("username", "walter2")
	form.Set("email", "walter@test.com") // same email
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

	// register a user first via the handler
	form := url.Values{}
	form.Set("username", "walter")
	form.Set("email", "walter@test.com")
	form.Set("password", "secret123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httptest.NewRecorder() // discard
	h.RegisterPOST(httptest.NewRecorder(), req)

	// now try to login
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

	// confirm a session cookie was set
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
}

func TestLoginPOST_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	h := NewAuthHandler(db)

	// register first
	form := url.Values{}
	form.Set("username", "walter")
	form.Set("email", "walter@test.com")
	form.Set("password", "secret123")
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.RegisterPOST(httptest.NewRecorder(), req)

	// login with wrong password
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
	h := NewAuthHandler(nil)

	// plant a fake session
	Sessions["fake-token-123"] = 1

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "fake-token-123"})

	rr := httptest.NewRecorder()
	h.Logout(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}

	// session should be gone
	if _, exists := Sessions["fake-token-123"]; exists {
		t.Error("expected session to be deleted after logout")
	}
}
