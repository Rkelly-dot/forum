package models

import "time"

type Post struct {
	ID         int64
	UserID     int64
	Username   string
	Title      string
	Body       string
	Categories []string
	Likes      int
	Dislikes   int
	CreatedAt  time.Time
}

type Category struct {
	ID   int64
	Name string
}

func (p Post) FormattedDate() string {
	eat := time.FixedZone("EAT", 3*60*60) // UTC+3
	return p.CreatedAt.In(eat).Format("Jan 2, 2006 at 3:04 PM")
}
