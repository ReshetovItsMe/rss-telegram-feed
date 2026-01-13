package service

import (
	"fmt"
	"log/slog"

	"github.com/gorilla/feeds"
	channelRepo "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/repository"
	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/domain"
	messageRepo "github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/repository"
	"github.com/samber/oops"
)

// Service handles RSS feed generation
type Service struct {
	channelRepo channelRepo.Repository
	messageRepo messageRepo.Repository
}

// New creates a new feed service
func New(channelRepo channelRepo.Repository, messageRepo messageRepo.Repository) *Service {
	return &Service{
		channelRepo: channelRepo,
		messageRepo: messageRepo,
	}
}

// GenerateFeed generates an RSS feed for a channel
func (s *Service) GenerateFeed(channelID string, baseURL string) (*feeds.Feed, error) {
	channel, err := s.channelRepo.GetChannel(channelID)
	if err != nil {
		return nil, oops.With("channel_id", channelID, "context", "channel not found").Wrap(err)
	}

	messages, err := s.messageRepo.GetMessages(channelID, 50) // Get last 50 messages
	if err != nil {
		return nil, oops.With("channel_id", channelID, "context", "failed to get messages").Wrap(err)
	}

	feed := &feeds.Feed{
		Title:       fmt.Sprintf("%s - RSS Feed", channel.Title),
		Link:        &feeds.Link{Href: fmt.Sprintf("%s/rss/%s", baseURL, channel.ID)},
		Description: fmt.Sprintf("RSS feed for Telegram channel: %s", channel.Title),
		Author:      &feeds.Author{Name: channel.Username},
		Created:     channel.AddedAt,
		Updated:     channel.LastUpdate,
	}

	var items []*feeds.Item
	for _, msg := range messages {
		item := s.messageToFeedItem(msg, baseURL)
		items = append(items, item)
	}

	feed.Items = items
	return feed, nil
}

// UpdateFeed triggers feed regeneration (placeholder for future caching)
func (s *Service) UpdateFeed(channelID string) error {
	// This method can be used to trigger feed regeneration
	// For now, feeds are generated on-demand when requested
	slog.Debug("Feed update requested", "channel_id", channelID)
	return nil
}

func (s *Service) messageToFeedItem(msg *domain.Message, baseURL string) *feeds.Item {
	description := msg.Text
	if description == "" {
		description = "No text content"
	}

	// Add media information to description
	if len(msg.Media) > 0 {
		description += "\n\nMedia:\n"
		for _, media := range msg.Media {
			description += fmt.Sprintf("- %s: %s\n", media.Type, media.FileID)
			if media.Caption != "" {
				description += fmt.Sprintf("  Caption: %s\n", media.Caption)
			}
		}
	}

	// Build content with HTML formatting for better RSS client compatibility
	content := fmt.Sprintf("<p>%s</p>", escapeHTML(description))
	if len(msg.Media) > 0 {
		content += "<p><strong>Media attachments:</strong></p><ul>"
		for _, media := range msg.Media {
			content += fmt.Sprintf("<li>%s: %s</li>", media.Type, media.FileID)
		}
		content += "</ul>"
	}

	item := &feeds.Item{
		Title:       truncate(msg.Text, 100),
		Link:        &feeds.Link{Href: msg.Link},
		Description: description,
		Content:     content,
		Author:      &feeds.Author{Name: msg.Author},
		Created:     msg.Date,
		Id:          fmt.Sprintf("%s-%d", msg.ChannelID, msg.ID),
	}

	return item
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func escapeHTML(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case '<':
			result = append(result, []rune("&lt;")...)
		case '>':
			result = append(result, []rune("&gt;")...)
		case '&':
			result = append(result, []rune("&amp;")...)
		case '"':
			result = append(result, []rune("&quot;")...)
		case '\'':
			result = append(result, []rune("&#39;")...)
		default:
			result = append(result, r)
		}
	}
	return string(result)
}
