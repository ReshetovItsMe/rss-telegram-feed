package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-telegram/bot"
	"github.com/reshetovitsme/rss-telegram-feed/internal/di"
	channelService "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/service"
	httpServer "github.com/reshetovitsme/rss-telegram-feed/internal/transport/http"
	"github.com/reshetovitsme/rss-telegram-feed/internal/shared/config"
	"github.com/samber/do/v2"
	slogmulti "github.com/samber/slog-multi"
)

func main() {
	// Setup structured logging with multiple handlers using slog-multi
	textHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	jsonHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	})

	// Use Fanout to send logs to both handlers
	multiHandler := slogmulti.Fanout(textHandler, jsonHandler)
	logger := slog.New(multiHandler)
	slog.SetDefault(logger)

	// Setup dependency injection
	injector, err := di.Setup()
	if err != nil {
		slog.Error("Failed to setup dependency injection", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := di.Shutdown(injector); err != nil {
			slog.Error("Error during shutdown", "error", err)
		}
	}()

	// Get services from DI container
	cfg := do.MustInvoke[*config.Config](injector)
	channelService := do.MustInvoke[*channelService.Service](injector)
	httpServer := do.MustInvoke[*httpServer.Server](injector)
	_ = do.MustInvoke[*bot.Bot](injector) // Initialize bot (already done in Setup)

	// Start channel monitoring
	go channelService.Start(context.Background())

	// Start HTTP server
	go func() {
		if err := httpServer.Start(); err != nil {
			slog.Error("Failed to start HTTP server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Application started", "port", cfg.HTTPPort)
	slog.Info("Press Ctrl+C to stop")

	// Graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	<-ctx.Done()
	slog.Info("Shutting down...")
}
