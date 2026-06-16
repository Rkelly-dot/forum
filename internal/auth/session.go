package auth

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"forum/internal/models"
)

func CreateSession(db *sql.DB, userID int) (*models.Session, error) {
	// delete any existing sessions for this user first
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
