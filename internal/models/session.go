package models

import "time"

type Session struct {
	ID        int64
	Token     string    
	UserID    int64       
	ExpiresAt time.Time 
	CreatedAt time.Time
}