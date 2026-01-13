package repository

import (
	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/user/domain"
)

// Repository defines the interface for user data persistence
type Repository interface {
	SaveUser(user *domain.User) error
	GetUser(userID int64) (*domain.User, error)
	GetAllUsers() ([]*domain.User, error)
}
