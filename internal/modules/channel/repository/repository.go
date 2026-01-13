package repository

import (
	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/domain"
)

// Repository defines the interface for channel data persistence
// This abstraction allows easy replacement of storage implementations
// (e.g., FileStorage -> PostgreSQL -> MongoDB)
type Repository interface {
	SaveChannel(channel *domain.Channel) error
	GetChannel(channelID string) (*domain.Channel, error)
	GetAllChannels() ([]*domain.Channel, error)
	DeleteChannel(channelID string) error
}
