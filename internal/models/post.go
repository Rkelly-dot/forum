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