package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/user/domain"
	"github.com/samber/oops"
)

// FileStorage implements user.Repository using file system
type FileStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileStorage creates a new file-based user repository
func NewFileStorage(basePath string) (Repository, error) {
	userPath := filepath.Join(basePath, "users")
	if err := os.MkdirAll(userPath, 0755); err != nil {
		return nil, oops.With("base_path", basePath, "context", "failed to create users directory").Wrap(err)
	}

	return &FileStorage{basePath: userPath}, nil
}

func (s *FileStorage) SaveUser(user *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, fmt.Sprintf("%d.json", user.ID))
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return oops.With("user_id", user.ID, "context", "failed to marshal user").Wrap(err)
	}

	return os.WriteFile(path, data, 0644)
}

func (s *FileStorage) GetUser(userID int64) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, fmt.Sprintf("%d.json", userID))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, oops.With("user_id", userID).New("user not found")
		}
		return nil, oops.With("user_id", userID, "context", "failed to read user").Wrap(err)
	}

	var user domain.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, oops.With("user_id", userID, "context", "failed to unmarshal user").Wrap(err)
	}

	return &user, nil
}

func (s *FileStorage) GetAllUsers() ([]*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, oops.With("directory", s.basePath, "context", "failed to read users directory").Wrap(err)
	}

	var users []*domain.User
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.basePath, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var user domain.User
		if err := json.Unmarshal(data, &user); err != nil {
			continue
		}

		users = append(users, &user)
	}

	return users, nil
}
