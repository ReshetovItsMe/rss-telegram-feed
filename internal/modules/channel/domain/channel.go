package domain

import "time"

// Channel represents a Telegram channel being monitored
type Channel struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Title       string    `json:"title"`
	AddedBy     int64     `json:"added_by"`
	AddedAt     time.Time `json:"added_at"`
	Filters     []Filter  `json:"filters"`
	LastUpdate  time.Time `json:"last_update"`
	IsActive    bool      `json:"is_active"`
}

// Filter represents content filtering criteria
type Filter struct {
	Type     FilterType `json:"type"`
	Keywords []string   `json:"keywords"`
	Enabled  bool       `json:"enabled"`
}
