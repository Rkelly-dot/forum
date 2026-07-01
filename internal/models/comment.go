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

func (c Comment) FormattedDate() string {
	eat := time.FixedZone("EAT", 3*60*60) // UTC+3
	return c.CreatedAt.In(eat).Format("Jan 2, 2006 at 3:04 PM")
}
