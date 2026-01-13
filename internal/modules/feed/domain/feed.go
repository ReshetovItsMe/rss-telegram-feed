package domain

import "time"

// FeedConfig represents RSS feed configuration
type FeedConfig struct {
	ChannelID string    `json:"channel_id"`
	Title     string    `json:"title"`
	Link      string    `json:"link"`
	Updated   time.Time `json:"updated"`
}
