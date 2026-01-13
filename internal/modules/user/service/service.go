package service

import (
	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/user/domain"
	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/user/repository"
)

// Service handles user business logic
type Service struct {
	repo repository.Repository
}

// New creates a new user service
func New(repo repository.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// SaveUser saves a user
func (s *Service) SaveUser(user *domain.User) error {
	return s.repo.SaveUser(user)
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(userID int64) (*domain.User, error) {
	return s.repo.GetUser(userID)
}

// GetAllUsers retrieves all users
func (s *Service) GetAllUsers() ([]*domain.User, error) {
	return s.repo.GetAllUsers()
}

// IsAuthorized checks if a user is authorized
func (s *Service) IsAuthorized(userID int64, allowedUsers []int64) bool {
	if len(allowedUsers) == 0 {
		return true // No restrictions
	}

	for _, id := range allowedUsers {
		if id == userID {
			return true
		}
	}

	return false
}
