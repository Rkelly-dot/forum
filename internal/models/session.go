package models

import "time"

// Session maps to the sessions table.
// Created on login, deleted on logout, checked on every protected request.
type Session struct {
	ID        int
	Token     string    // UUID stored in the browser cookie
	UserID    int       // FK → users.id
	ExpiresAt time.Time // handler rejects sessions past this time
	CreatedAt time.Time
}