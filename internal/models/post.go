package models

import "time"

// Post maps to the posts table.
// Categories is populated by a JOIN on post_categories — not stored in posts table directly.
type Post struct {
	ID         int
	UserID     int
	Username   string     // joined from users table for display
	Title      string
	Content    string
	Categories []Category // joined from categories via post_categories
	Likes      int        // aggregated from likes table
	Dislikes   int        // aggregated from likes table
	CreatedAt  time.Time
}

// Category maps to the categories table.
type Category struct {
	ID   int
	Name string
}