package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/domain"
	channelRepo "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/repository"
	messageDomain "github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/domain"
	messageRepo "github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/repository"
	"github.com/reshetovitsme/rss-telegram-feed/internal/shared/config"
	"github.com/samber/oops"
)

// Service handles channel business logic
type Service struct {
	cfg         *config.Config
	channelRepo channelRepo.Repository
	messageRepo messageRepo.Repository
	bot         *bot.Bot
	channels    map[string]bool
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// New creates a new channel service
func New(cfg *config.Config, channelRepo channelRepo.Repository, messageRepo messageRepo.Repository) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		cfg:         cfg,
		channelRepo: channelRepo,
		messageRepo: messageRepo,
		channels:    make(map[string]bool),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetBot sets the Telegram bot instance
func (s *Service) SetBot(b *bot.Bot) {
	s.bot = b
}

// Start begins monitoring channels
func (s *Service) Start(ctx context.Context) {
	// Load existing channels
	channels, err := s.channelRepo.GetAllChannels()
	if err != nil {
		slog.Error("Failed to load channels", "error", err)
	} else {
		for _, ch := range channels {
			if ch.IsActive {
				s.AddChannel(ch.ID)
			}
		}
	}

	// Start monitoring loop
	s.wg.Add(1)
	go s.monitorLoop()
}

// Stop stops monitoring
func (s *Service) Stop() {
	s.cancel()
	s.wg.Wait()
}

// AddChannel adds a channel to monitoring
func (s *Service) AddChannel(channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channels[channelID] = true
}

// RemoveChannel removes a channel from monitoring
func (s *Service) RemoveChannel(channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.channels, channelID)
}

// GetChannel retrieves a channel by ID
func (s *Service) GetChannel(channelID string) (*domain.Channel, error) {
	return s.channelRepo.GetChannel(channelID)
}

// GetAllChannels retrieves all channels
func (s *Service) GetAllChannels() ([]*domain.Channel, error) {
	return s.channelRepo.GetAllChannels()
}

// SaveChannel saves a channel
func (s *Service) SaveChannel(channel *domain.Channel) error {
	return s.channelRepo.SaveChannel(channel)
}

// DeleteChannel deletes a channel
func (s *Service) DeleteChannel(channelID string) error {
	s.RemoveChannel(channelID)
	return s.channelRepo.DeleteChannel(channelID)
}

// ProcessMessage processes a message from a channel
func (s *Service) ProcessMessage(channel *domain.Channel, msgText string, msgDate int64, msgID int64, author string, media []messageDomain.Media, link string) error {
	// Apply filters
	if !s.passesFilters(channel, msgText) {
		return nil
	}

	// Convert to message domain
	message := &messageDomain.Message{
		ID:          msgID,
		ChannelID:   channel.ID,
		ChannelName: channel.Title,
		Text:        msgText,
		Date:        time.Unix(msgDate, 0),
		Author:      author,
		Media:       media,
		Link:        link,
	}

	// Save message
	if err := s.messageRepo.SaveMessage(message); err != nil {
		return oops.With("channel_id", channel.ID, "message_id", message.ID, "context", "failed to save message").Wrap(err)
	}

	// Update channel last update time
	channel.LastUpdate = time.Now()
	if err := s.channelRepo.SaveChannel(channel); err != nil {
		slog.Error("Failed to update channel last update time", "channel_id", channel.ID, "error", err)
	}

	return nil
}

// passesFilters checks if a message passes all enabled filters
func (s *Service) passesFilters(channel *domain.Channel, text string) bool {
	if len(channel.Filters) == 0 {
		return true
	}

	for _, filter := range channel.Filters {
		if !filter.Enabled {
			continue
		}

		switch filter.Type {
		case domain.FilterTypeKeywords:
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
		case domain.FilterTypeExcludeKeywords:
			// Check if any exclude keyword is present
			for _, keyword := range filter.Keywords {
				if contains(text, keyword) {
					return false
				}
			}
		case domain.FilterTypeAuthor:
			// Author filtering can be implemented here if needed
		}
	}

	return true
}

func (s *Service) monitorLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Duration(s.cfg.UpdateInterval) * time.Second)
	defer ticker.Stop()

	// Initial check
	s.checkChannels()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkChannels()
		}
	}
}

func (s *Service) checkChannels() {
	s.mu.RLock()
	channelIDs := make([]string, 0, len(s.channels))
	for id := range s.channels {
		channelIDs = append(channelIDs, id)
	}
	s.mu.RUnlock()

	for _, channelID := range channelIDs {
		s.wg.Add(1)
		go func(id string) {
			defer s.wg.Done()
			if err := s.fetchChannelMessages(id); err != nil {
				slog.Error("Error fetching messages for channel", "channel_id", id, "error", err)
			}
		}(channelID)
	}
}

func (s *Service) fetchChannelMessages(channelID string) error {
	if s.bot == nil {
		return oops.Errorf("bot not initialized")
	}

	channel, err := s.channelRepo.GetChannel(channelID)
	if err != nil {
		return oops.With("channel_id", channelID, "context", "failed to get channel").Wrap(err)
	}

	if !channel.IsActive {
		return nil
	}

	// Note: Channel messages are processed in real-time via the bot's HandleUpdate
	// This method is kept for potential future enhancements (e.g., backfilling history)
	slog.Debug("Channel monitoring check", "channel_id", channelID, "note", "messages processed via real-time updates")

	// Update last check time
	channel.LastUpdate = time.Now()
	if err := s.channelRepo.SaveChannel(channel); err != nil {
		slog.Error("Failed to update channel last update time", "channel_id", channelID, "error", err)
	}

	return nil
}

// Helper functions
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
