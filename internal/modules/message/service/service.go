package service

import (
	"time"

	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/domain"
	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/repository"
)

// Service handles message business logic
type Service struct {
	repo repository.Repository
}

// New creates a new message service
func New(repo repository.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// SaveMessage saves a message
func (s *Service) SaveMessage(message *domain.Message) error {
	return s.repo.SaveMessage(message)
}

// GetMessages retrieves messages for a channel
func (s *Service) GetMessages(channelID string, limit int) ([]*domain.Message, error) {
	return s.repo.GetMessages(channelID, limit)
}

// GetRecentMessages retrieves recent messages since a given time
func (s *Service) GetRecentMessages(channelID string, since time.Time) ([]*domain.Message, error) {
	return s.repo.GetRecentMessages(channelID, since)
}
