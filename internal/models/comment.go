package models

import "time"

type Comment struct {
	ID        int64
	PostID    int64
	UserID    int64
	Username  string
	Body      string
	Likes     int
	Dislikes  int
	CreatedAt time.Time
}
