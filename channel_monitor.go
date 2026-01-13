package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/samber/oops"
)

type ChannelMonitor struct {
	cfg         *Config
	storage     Storage
	feedService *FeedService
	bot         *bot.Bot
	channels    map[string]bool
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewChannelMonitor(cfg *Config, storage Storage, feedService *FeedService) *ChannelMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &ChannelMonitor{
		cfg:         cfg,
		storage:     storage,
		feedService: feedService,
		channels:    make(map[string]bool),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (m *ChannelMonitor) Start(ctx context.Context) {
	// Load existing channels
	channels, err := m.storage.GetAllChannels()
	if err != nil {
		slog.Error("Failed to load channels", "error", err)
	} else {
		for _, ch := range channels {
			if ch.IsActive {
				m.AddChannel(ch.ID)
			}
		}
	}

	// Start monitoring loop
	m.wg.Add(1)
	go m.monitorLoop()
}

func (m *ChannelMonitor) Stop() {
	m.cancel()
	m.wg.Wait()
}

func (m *ChannelMonitor) AddChannel(channelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[channelID] = true
}

func (m *ChannelMonitor) RemoveChannel(channelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.channels, channelID)
}

func (m *ChannelMonitor) SetBot(b *bot.Bot) {
	m.bot = b
}

func (m *ChannelMonitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(time.Duration(m.cfg.UpdateInterval) * time.Second)
	defer ticker.Stop()

	// Initial check
	m.checkChannels()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkChannels()
		}
	}
}

func (m *ChannelMonitor) checkChannels() {
	m.mu.RLock()
	channelIDs := make([]string, 0, len(m.channels))
	for id := range m.channels {
		channelIDs = append(channelIDs, id)
	}
	m.mu.RUnlock()

	for _, channelID := range channelIDs {
		m.wg.Add(1)
		go func(id string) {
			defer m.wg.Done()
			if err := m.fetchChannelMessages(id); err != nil {
				slog.Error("Error fetching messages for channel", "channel_id", id, "error", err)
			}
		}(channelID)
	}
}

func (m *ChannelMonitor) fetchChannelMessages(channelID string) error {
	if m.bot == nil {
		return oops.Errorf("bot not initialized")
	}

	channel, err := m.storage.GetChannel(channelID)
	if err != nil {
		return oops.With("channel_id", channelID, "context", "failed to get channel").Wrap(err)
	}

	if !channel.IsActive {
		return nil
	}

	// Note: Channel messages are processed in real-time via the bot's HandleUpdate
	// This method is kept for potential future enhancements (e.g., backfilling history)
	// For now, messages are captured as they arrive through the bot handler
	slog.Debug("Channel monitoring check", "channel_id", channelID, "note", "messages processed via real-time updates")

	// Update last check time
	channel.LastUpdate = time.Now()
	if err := m.storage.SaveChannel(channel); err != nil {
		slog.Error("Failed to update channel last update time", "channel_id", channelID, "error", err)
	}

	return nil
}

func (m *ChannelMonitor) processMessage(channel *Channel, msg *models.Message) error {
	// Apply filters
	if !m.passesFilters(channel, msg) {
		return nil
	}

	// Convert Telegram message to our Message model
	message := &Message{
		ID:          int64(msg.ID),
		ChannelID:   channel.ID,
		ChannelName: channel.Title,
		Text:        msg.Text,
		Date:        time.Unix(int64(msg.Date), 0),
		Author:      getAuthorName(msg),
		Media:       extractMedia(msg),
		Link:        fmt.Sprintf("https://t.me/%s/%d", channel.Username, msg.ID),
	}

	// Save message
	if err := m.storage.SaveMessage(message); err != nil {
		return oops.With("channel_id", channel.ID, "message_id", message.ID, "context", "failed to save message").Wrap(err)
	}

	// Update RSS feed
	if err := m.feedService.UpdateFeed(channel.ID); err != nil {
		slog.Error("Failed to update RSS feed", "channel_id", channel.ID, "error", err)
	}

	return nil
}

func (m *ChannelMonitor) passesFilters(channel *Channel, msg *models.Message) bool {
	if len(channel.Filters) == 0 {
		return true
	}

	text := msg.Text
	if text == "" && msg.Caption != "" {
		text = msg.Caption
	}

	for _, filter := range channel.Filters {
		if !filter.Enabled {
			continue
		}

		switch filter.Type {
		case FilterTypeKeywords:
			// Check if any keyword is present
			matches := false
			for _, keyword := range filter.Keywords {
				if contains(text, keyword) {
					matches = true
					break
				}
			}
			if !matches {
				return false
			}
		case FilterTypeExcludeKeywords:
			// Check if any exclude keyword is present
			for _, keyword := range filter.Keywords {
				if contains(text, keyword) {
					return false
				}
			}
		case FilterTypeAuthor:
			// Author filtering can be implemented here if needed
		}
	}

	return true
}

func contains(text, keyword string) bool {
	return len(keyword) > 0 && len(text) >= len(keyword) &&
		(text == keyword || containsSubstring(text, keyword))
}

func containsSubstring(text, substr string) bool {
	textLower := toLower(text)
	substrLower := toLower(substr)
	for i := 0; i <= len(textLower)-len(substrLower); i++ {
		if textLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// getAuthorName and extractMedia are defined in bot_handler.go
