package models

import "time"


type User struct {
	ID        int64
	Email     string
	Username  string
	Password  string 
	CreatedAt time.Time
}
