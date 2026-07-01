package auth

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"forum/internal/models"
)

func CreateSession(db *sql.DB, userID int64) (*models.Session, error) {
	_, err := db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("could not clear old sessions: %w", err)
	}

	session := &models.Session{
		Token:     uuid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	_, err = db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		session.Token, session.UserID, session.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create session: %w", err)
	}

	return session, nil
}

func GetSessionUser(r *http.Request, db *sql.DB) (*models.User, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil, nil
	}
	session, err := ValidateSession(db, cookie.Value)
	if err != nil {
		return nil, nil
	}
	var user models.User
	err = db.QueryRow(
		"SELECT id, username, email, password, created_at FROM users WHERE id = ?",
		session.UserID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func ValidateSession(db *sql.DB, token string) (*models.Session, error) {
	var session models.Session
	err := db.QueryRow(
		"SELECT user_id, expires_at FROM sessions WHERE token = ?", token,
	).Scan(&session.UserID, &session.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		db.Exec("DELETE FROM sessions WHERE token = ?", token)
		return nil, fmt.Errorf("session expired")
	}

	session.Token = token
	return &session, nil
}

func DeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}
