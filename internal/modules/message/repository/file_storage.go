package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/domain"
	"github.com/samber/oops"
)

// FileStorage implements message.Repository using file system
type FileStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileStorage creates a new file-based message repository
func NewFileStorage(basePath string) (Repository, error) {
	messagePath := filepath.Join(basePath, "messages")
	if err := os.MkdirAll(messagePath, 0755); err != nil {
		return nil, oops.With("base_path", basePath, "context", "failed to create messages directory").Wrap(err)
	}

	return &FileStorage{basePath: messagePath}, nil
}

func (s *FileStorage) SaveMessage(message *domain.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store messages in channel-specific directories
	msgDir := filepath.Join(s.basePath, message.ChannelID)
	if err := os.MkdirAll(msgDir, 0755); err != nil {
		return oops.With("message_dir", msgDir, "context", "failed to create message directory").Wrap(err)
	}

	path := filepath.Join(msgDir, fmt.Sprintf("%d.json", message.ID))
	data, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return oops.With("channel_id", message.ChannelID, "message_id", message.ID, "context", "failed to marshal message").Wrap(err)
	}

	return os.WriteFile(path, data, 0644)
}

func (s *FileStorage) GetMessages(channelID string, limit int) ([]*domain.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgDir := filepath.Join(s.basePath, channelID)
	entries, err := os.ReadDir(msgDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*domain.Message{}, nil
		}
		return nil, oops.With("channel_id", channelID, "message_dir", msgDir, "context", "failed to read messages directory").Wrap(err)
	}

	var messages []*domain.Message
	count := 0
	for i := len(entries) - 1; i >= 0 && count < limit; i-- {
		entry := entries[i]
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(msgDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var message domain.Message
		if err := json.Unmarshal(data, &message); err != nil {
			continue
		}

		messages = append(messages, &message)
		count++
	}

	return messages, nil
}

func (s *FileStorage) GetRecentMessages(channelID string, since time.Time) ([]*domain.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgDir := filepath.Join(s.basePath, channelID)
	entries, err := os.ReadDir(msgDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*domain.Message{}, nil
		}
		return nil, oops.With("channel_id", channelID, "message_dir", msgDir, "context", "failed to read messages directory").Wrap(err)
	}

	var messages []*domain.Message
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(msgDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var message domain.Message
		if err := json.Unmarshal(data, &message); err != nil {
			continue
		}

		if message.Date.After(since) {
			messages = append(messages, &message)
		}
	}

	return messages, nil
}
