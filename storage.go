package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/samber/lo"
)

type Storage interface {
	SaveChannel(channel *Channel) error
	GetChannel(channelID string) (*Channel, error)
	GetAllChannels() ([]*Channel, error)
	DeleteChannel(channelID string) error
	SaveMessage(message *Message) error
	GetMessages(channelID string, limit int) ([]*Message, error)
	GetRecentMessages(channelID string, since time.Time) ([]*Message, error)
	SaveUser(user *User) error
	GetUser(userID int64) (*User, error)
	GetAllUsers() ([]*User, error)
}

type FileStorage struct {
	basePath string
	mu       sync.RWMutex
}

func NewFileStorage(basePath string) (*FileStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create subdirectories using lo
	dirs := []string{"channels", "messages", "users"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(basePath, dir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	return &FileStorage{basePath: basePath}, nil
}

func (s *FileStorage) SaveChannel(channel *Channel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "channels", channel.ID+".json")
	data, err := json.MarshalIndent(channel, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal channel: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func (s *FileStorage) GetChannel(channelID string) (*Channel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "channels", channelID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrChannelNotFound
		}
		return nil, fmt.Errorf("failed to read channel: %w", err)
	}

	var channel Channel
	if err := json.Unmarshal(data, &channel); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channel: %w", err)
	}

	return &channel, nil
}

func (s *FileStorage) GetAllChannels() ([]*Channel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.basePath, "channels")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read channels directory: %w", err)
	}

	// Use lo.FilterMap to process entries
	channels := lo.FilterMap(entries, func(entry os.DirEntry, _ int) (*Channel, bool) {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil, false
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, false
		}

		var channel Channel
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

	path := filepath.Join(s.basePath, "channels", channelID+".json")
	return os.Remove(path)
}

func (s *FileStorage) SaveMessage(message *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store messages in channel-specific directories
	msgDir := filepath.Join(s.basePath, "messages", message.ChannelID)
	if err := os.MkdirAll(msgDir, 0755); err != nil {
		return fmt.Errorf("failed to create message directory: %w", err)
	}

	path := filepath.Join(msgDir, fmt.Sprintf("%d.json", message.ID))
	data, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func (s *FileStorage) GetMessages(channelID string, limit int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgDir := filepath.Join(s.basePath, "messages", channelID)
	entries, err := os.ReadDir(msgDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Message{}, nil
		}
		return nil, fmt.Errorf("failed to read messages directory: %w", err)
	}

	var messages []*Message
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

		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			continue
		}

		messages = append(messages, &message)
		count++
	}

	return messages, nil
}

func (s *FileStorage) GetRecentMessages(channelID string, since time.Time) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgDir := filepath.Join(s.basePath, "messages", channelID)
	entries, err := os.ReadDir(msgDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Message{}, nil
		}
		return nil, fmt.Errorf("failed to read messages directory: %w", err)
	}

	var messages []*Message
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(msgDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			continue
		}

		if message.Date.After(since) {
			messages = append(messages, &message)
		}
	}

	return messages, nil
}

func (s *FileStorage) SaveUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.basePath, "users", fmt.Sprintf("%d.json", user.ID))
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func (s *FileStorage) GetUser(userID int64) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.basePath, "users", fmt.Sprintf("%d.json", userID))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to read user: %w", err)
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (s *FileStorage) GetAllUsers() ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.basePath, "users")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read users directory: %w", err)
	}

	var users []*User
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			continue
		}

		users = append(users, &user)
	}

	return users, nil
}
