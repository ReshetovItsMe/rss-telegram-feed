package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/domain"
	"github.com/reshetovitsme/rss-telegram-feed/internal/shared/errors"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

// FileStorage implements channel.Repository using file system
type FileStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileStorage creates a new file-based channel repository
func NewFileStorage(basePath string) (Repository, error) {
	channelPath := filepath.Join(basePath, "channels")
	if err := os.MkdirAll(channelPath, 0755); err != nil {
		return nil, oops.With("base_path", basePath, "context", "failed to create channels directory").Wrap(err)
	}

	return &FileStorage{basePath: channelPath}, nil
}

func (s *FileStorage) SaveChannel(channel *domain.Channel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, channel.ID+".json")
	data, err := json.MarshalIndent(channel, "", "  ")
	if err != nil {
		return oops.With("channel_id", channel.ID, "context", "failed to marshal channel").Wrap(err)
	}

	return os.WriteFile(path, data, 0644)
}

func (s *FileStorage) GetChannel(channelID string) (*domain.Channel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, channelID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.ErrChannelNotFound
		}
		return nil, oops.With("channel_id", channelID, "context", "failed to read channel").Wrap(err)
	}

	var channel domain.Channel
	if err := json.Unmarshal(data, &channel); err != nil {
		return nil, oops.With("channel_id", channelID, "context", "failed to unmarshal channel").Wrap(err)
	}

	return &channel, nil
}

func (s *FileStorage) GetAllChannels() ([]*domain.Channel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, oops.With("directory", s.basePath, "context", "failed to read channels directory").Wrap(err)
	}

	channels := lo.FilterMap(entries, func(entry os.DirEntry, _ int) (*domain.Channel, bool) {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil, false
		}

		path := filepath.Join(s.basePath, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, false
		}

		var channel domain.Channel
		if err := json.Unmarshal(data, &channel); err != nil {
			return nil, false
		}

		return &channel, true
	})

	return channels, nil
}

func (s *FileStorage) DeleteChannel(channelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, channelID+".json")
	return os.Remove(path)
}
