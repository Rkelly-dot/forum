package models

import "time"

// Comment maps to the comments table.
// Username is populated by a JOIN on users — not stored in comments table directly.
type Comment struct {
	ID        int
	PostID    int
	UserID    int
	Username  string // joined from users table for display
	Content   string
	Likes     int // aggregated from likes table
	Dislikes  int // aggregated from likes table
	CreatedAt time.Time
}