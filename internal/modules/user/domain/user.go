package domain

import "time"

// User represents an authorized user
type User struct {
	ID       int64     `json:"id"`
	Username string    `json:"username"`
	AddedAt  time.Time `json:"added_at"`
	IsAdmin  bool      `json:"is_admin"`
}
