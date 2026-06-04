package models

import "time"

// User maps directly to the users table.
// Shared by auth, post, and comment packages.
type User struct {
	ID        int
	Email     string
	Username  string
	Password  string // bcrypt hash — never the raw password
	CreatedAt time.Time
}