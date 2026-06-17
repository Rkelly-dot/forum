package models

import "time"

type Session struct {
	ID        int
	Token     string    
	UserID    int       
	ExpiresAt time.Time 
	CreatedAt time.Time
}