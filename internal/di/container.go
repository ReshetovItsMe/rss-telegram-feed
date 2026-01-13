package di

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	channelRepo "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/repository"
	channelService "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/service"
	feedService "github.com/reshetovitsme/rss-telegram-feed/internal/modules/feed/service"
	messageRepo "github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/repository"
	messageService "github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/service"
	userRepo "github.com/reshetovitsme/rss-telegram-feed/internal/modules/user/repository"
	userService "github.com/reshetovitsme/rss-telegram-feed/internal/modules/user/service"
	"github.com/reshetovitsme/rss-telegram-feed/internal/shared/config"
	telegramHandler "github.com/reshetovitsme/rss-telegram-feed/internal/transport/telegram"
	httpServer "github.com/reshetovitsme/rss-telegram-feed/internal/transport/http"
	"github.com/samber/do/v2"
	"github.com/samber/oops"
)

// Service names for dependency injection
const (
	ServiceConfig         = "config"
	ServiceChannelRepo    = "channel-repository"
	ServiceMessageRepo    = "message-repository"
	ServiceUserRepo       = "user-repository"
	ServiceChannelService = "channel-service"
	ServiceMessageService = "message-service"
	ServiceUserService    = "user-service"
	ServiceFeedService    = "feed-service"
	ServiceTelegramHandler = "telegram-handler"
	ServiceHTTPServer     = "http-server"
	ServiceBot            = "bot"
)

// Setup initializes the dependency injection container
func Setup() (do.Injector, error) {
	injector := do.New()

	// Register Config
	do.Provide(injector, func(i do.Injector) (*config.Config, error) {
		cfg, err := config.Load()
		if err != nil {
			return nil, oops.With("context", "failed to load config").Wrap(err)
		}
		return cfg, nil
	})

	// Register Channel Repository
	do.Provide(injector, func(i do.Injector) (channelRepo.Repository, error) {
		cfg := do.MustInvoke[*config.Config](i)
		repo, err := channelRepo.NewFileStorage(cfg.StoragePath)
		if err != nil {
			return nil, oops.With("storage_path", cfg.StoragePath, "context", "failed to initialize channel repository").Wrap(err)
		}
		return repo, nil
	})

	// Register Message Repository
	do.Provide(injector, func(i do.Injector) (messageRepo.Repository, error) {
		cfg := do.MustInvoke[*config.Config](i)
		repo, err := messageRepo.NewFileStorage(cfg.StoragePath)
		if err != nil {
			return nil, oops.With("storage_path", cfg.StoragePath, "context", "failed to initialize message repository").Wrap(err)
		}
		return repo, nil
	})

	// Register User Repository
	do.Provide(injector, func(i do.Injector) (userRepo.Repository, error) {
		cfg := do.MustInvoke[*config.Config](i)
		repo, err := userRepo.NewFileStorage(cfg.StoragePath)
		if err != nil {
			return nil, oops.With("storage_path", cfg.StoragePath, "context", "failed to initialize user repository").Wrap(err)
		}
		return repo, nil
	})

	// Register Message Service
	do.Provide(injector, func(i do.Injector) (*messageService.Service, error) {
		repo := do.MustInvoke[messageRepo.Repository](i)
		return messageService.New(repo), nil
	})

	// Register User Service
	do.Provide(injector, func(i do.Injector) (*userService.Service, error) {
		repo := do.MustInvoke[userRepo.Repository](i)
		return userService.New(repo), nil
	})

	// Register Channel Service
	do.Provide(injector, func(i do.Injector) (*channelService.Service, error) {
		cfg := do.MustInvoke[*config.Config](i)
		chRepo := do.MustInvoke[channelRepo.Repository](i)
		msgRepo := do.MustInvoke[messageRepo.Repository](i)
		return channelService.New(cfg, chRepo, msgRepo), nil
	})

	// Register Feed Service
	do.Provide(injector, func(i do.Injector) (*feedService.Service, error) {
		chRepo := do.MustInvoke[channelRepo.Repository](i)
		msgRepo := do.MustInvoke[messageRepo.Repository](i)
		return feedService.New(chRepo, msgRepo), nil
	})

	// Register Telegram Handler
	do.Provide(injector, func(i do.Injector) (*telegramHandler.Handler, error) {
		cfg := do.MustInvoke[*config.Config](i)
		channelService := do.MustInvoke[*channelService.Service](i)
		feedService := do.MustInvoke[*feedService.Service](i)
		userService := do.MustInvoke[*userService.Service](i)
		return telegramHandler.New(cfg, channelService, feedService, userService), nil
	})

	// Register HTTP Server
	do.Provide(injector, func(i do.Injector) (*httpServer.Server, error) {
		cfg := do.MustInvoke[*config.Config](i)
		feedService := do.MustInvoke[*feedService.Service](i)
		server := httpServer.New(cfg, feedService)
		server.SetLogger(slog.Default())
		return server, nil
	})

	// Register Bot (needs to be initialized after handlers are ready)
	do.Provide(injector, func(i do.Injector) (*bot.Bot, error) {
		cfg := do.MustInvoke[*config.Config](i)
		telegramHandler := do.MustInvoke[*telegramHandler.Handler](i)

		opts := []bot.Option{
			bot.WithDefaultHandler(telegramHandler.HandleUpdate),
		}

		b, err := bot.New(cfg.TelegramBotToken, opts...)
		if err != nil {
			return nil, oops.With("context", "failed to create telegram bot").Wrap(err)
		}

		// Register bot commands
		telegramHandler.RegisterCommands(b)

		// Set bot in channel service
		channelService := do.MustInvoke[*channelService.Service](i)
		channelService.SetBot(b)

		return b, nil
	})

	return injector, nil
}

// Shutdown gracefully shuts down all services
func Shutdown(injector do.Injector) error {
	ctx := context.Background()

	// Shutdown bot if it exists
	if b, err := do.Invoke[*bot.Bot](injector); err == nil && b != nil {
		b.Close(ctx)
	}

	// Shutdown channel service if it exists
	if channelService, err := do.Invoke[*channelService.Service](injector); err == nil && channelService != nil {
		channelService.Stop()
	}

	return nil
}
