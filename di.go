package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/samber/do/v2"
)

// Service names for dependency injection
const (
	ServiceConfig         = "config"
	ServiceStorage        = "storage"
	ServiceFeedService    = "feed-service"
	ServiceChannelMonitor = "channel-monitor"
	ServiceBotHandler     = "bot-handler"
	ServiceRSSServer      = "rss-server"
	ServiceBot            = "bot"
)

// SetupDI initializes the dependency injection container
func SetupDI() (do.Injector, error) {
	injector := do.New()

	// Register Config
	do.Provide(injector, func(i do.Injector) (*Config, error) {
		cfg, err := LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		return cfg, nil
	})

	// Register Storage
	do.Provide(injector, func(i do.Injector) (Storage, error) {
		cfg := do.MustInvoke[*Config](i)
		storage, err := NewFileStorage(cfg.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize storage at %s: %w", cfg.StoragePath, err)
		}
		return storage, nil
	})

	// Register FeedService
	do.Provide(injector, func(i do.Injector) (*FeedService, error) {
		storage := do.MustInvoke[Storage](i)
		return NewFeedService(storage), nil
	})

	// Register ChannelMonitor
	do.Provide(injector, func(i do.Injector) (*ChannelMonitor, error) {
		cfg := do.MustInvoke[*Config](i)
		storage := do.MustInvoke[Storage](i)
		feedService := do.MustInvoke[*FeedService](i)
		return NewChannelMonitor(cfg, storage, feedService), nil
	})

	// Register BotHandler
	do.Provide(injector, func(i do.Injector) (*BotHandler, error) {
		cfg := do.MustInvoke[*Config](i)
		storage := do.MustInvoke[Storage](i)
		channelMonitor := do.MustInvoke[*ChannelMonitor](i)
		feedService := do.MustInvoke[*FeedService](i)
		return NewBotHandler(cfg, storage, channelMonitor, feedService), nil
	})

	// Register RSSServer
	do.Provide(injector, func(i do.Injector) (*RSSServer, error) {
		cfg := do.MustInvoke[*Config](i)
		feedService := do.MustInvoke[*FeedService](i)
		server := NewRSSServer(cfg, feedService)
		// Set logger from default slog
		server.SetLogger(slog.Default())
		return server, nil
	})

	// Register Bot (needs to be initialized after handlers are ready)
	do.Provide(injector, func(i do.Injector) (*bot.Bot, error) {
		cfg := do.MustInvoke[*Config](i)
		botHandler := do.MustInvoke[*BotHandler](i)

		opts := []bot.Option{
			bot.WithDefaultHandler(botHandler.HandleUpdate),
		}

		b, err := bot.New(cfg.TelegramBotToken, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create telegram bot: %w", err)
		}

		// Register bot commands
		botHandler.RegisterCommands(b)

		// Set bot in channel monitor
		channelMonitor := do.MustInvoke[*ChannelMonitor](i)
		channelMonitor.SetBot(b)

		return b, nil
	})

	return injector, nil
}

// ShutdownDI gracefully shuts down all services
func ShutdownDI(injector do.Injector) error {
	ctx := context.Background()

	// Shutdown bot if it exists
	if b, err := do.Invoke[*bot.Bot](injector); err == nil && b != nil {
		b.Close(ctx)
	}

	// Shutdown channel monitor if it exists
	if monitor, err := do.Invoke[*ChannelMonitor](injector); err == nil && monitor != nil {
		monitor.Stop()
	}

	return nil
}
