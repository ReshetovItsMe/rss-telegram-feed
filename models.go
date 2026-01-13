package main

import (
	"time"
)

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
	Type     FilterType `json:"type"` // FilterType enum
	Keywords []string   `json:"keywords"`
	Enabled  bool       `json:"enabled"`
}

// Message represents a Telegram message stored for RSS feed
type Message struct {
	ID          int64     `json:"id"`
	ChannelID   string    `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	Text        string    `json:"text"`
	Date        time.Time `json:"date"`
	Author      string    `json:"author"`
	Media       []Media   `json:"media"`
	Link        string    `json:"link"`
}

// Media represents multimedia content in a message
type Media struct {
	Type      MediaType `json:"type"` // MediaType enum
	FileID    string    `json:"file_id"`
	URL       string    `json:"url"`
	Thumbnail string    `json:"thumbnail,omitempty"`
	Caption   string    `json:"caption,omitempty"`
}

// User represents an authorized user
type User struct {
	ID       int64     `json:"id"`
	Username string    `json:"username"`
	AddedAt  time.Time `json:"added_at"`
	IsAdmin  bool      `json:"is_admin"`
}

// FeedConfig represents RSS feed configuration
type FeedConfig struct {
	ChannelID string    `json:"channel_id"`
	Title     string    `json:"title"`
	Link      string    `json:"link"`
	Updated   time.Time `json:"updated"`
}
