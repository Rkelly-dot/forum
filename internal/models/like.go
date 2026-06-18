package models

import "time"

// Like maps to the likes table.
// Value is always +1 (like) or -1 (dislike).
// Either PostID or CommentID is set — never both, never neither.
// This constraint is also enforced at the DB level in migrations.go.
type Like struct {
	ID        int
	UserID    int
	PostID    *int // pointer — nil when this like targets a comment
	CommentID *int // pointer — nil when this like targets a post
	Value     int  // +1 or -1
	CreatedAt time.Time
}
