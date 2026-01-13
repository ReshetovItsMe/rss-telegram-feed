package repository

import (
	"time"

	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/domain"
)

// Repository defines the interface for message data persistence
type Repository interface {
	SaveMessage(message *domain.Message) error
	GetMessages(channelID string, limit int) ([]*domain.Message, error)
	GetRecentMessages(channelID string, since time.Time) ([]*domain.Message, error)
}
