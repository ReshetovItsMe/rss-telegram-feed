package domain

import "time"

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
	Type      MediaType `json:"type"`
	FileID    string    `json:"file_id"`
	URL       string    `json:"url"`
	Thumbnail string    `json:"thumbnail,omitempty"`
	Caption   string    `json:"caption,omitempty"`
}
