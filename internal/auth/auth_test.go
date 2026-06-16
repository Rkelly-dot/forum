package auth

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

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

func TestRegisterUser_Success(t *testing.T) {
	db := setupTestDB(t)

	err := RegisterUser(db, "walter", "walter@test.com", "secret123")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "walter@test.com").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 user in db, got %d", count)
	}
}

func TestRegisterUser_MissingFields(t *testing.T) {
	db := setupTestDB(t)

	err := RegisterUser(db, "walter", "", "")
	if err == nil {
		t.Error("expected error for missing fields")
	}
}

func TestRegisterUser_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)

	RegisterUser(db, "walter", "walter@test.com", "secret123")
	err := RegisterUser(db, "walter2", "walter@test.com", "anotherpass")
	if err == nil {
		t.Error("expected error for duplicate email")
	}
}

func TestLoginUser_Success(t *testing.T) {
	db := setupTestDB(t)

	RegisterUser(db, "walter", "walter@test.com", "secret123")

	user, err := LoginUser(db, "walter@test.com", "secret123")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if user.Email != "walter@test.com" {
		t.Errorf("expected email walter@test.com, got %s", user.Email)
	}
}

func TestLoginUser_WrongPassword(t *testing.T) {
	db := setupTestDB(t)

	RegisterUser(db, "walter", "walter@test.com", "secret123")

	_, err := LoginUser(db, "walter@test.com", "wrongpassword")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestLoginUser_NonexistentUser(t *testing.T) {
	db := setupTestDB(t)

	_, err := LoginUser(db, "nobody@test.com", "whatever")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestDeleteSession(t *testing.T) {
	db := setupTestDB(t)

	RegisterUser(db, "walter", "walter@test.com", "secret123")
	user, _ := LoginUser(db, "walter@test.com", "secret123")
	session, _ := CreateSession(db, user.ID)

	err := DeleteSession(db, session.Token)
	if err != nil {
		t.Errorf("expected no error on delete, got %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", session.Token).Scan(&count)
	if count != 0 {
		t.Errorf("expected session to be deleted, got %d rows", count)
	}
}
