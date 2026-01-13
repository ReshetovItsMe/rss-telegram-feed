package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-telegram/bot"
	"github.com/samber/do/v2"
	slogmulti "github.com/samber/slog-multi"
)

func main() {
	// Setup structured logging with multiple handlers using slog-multi
	// Fanout sends logs to multiple handlers simultaneously
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
	injector, err := SetupDI()
	if err != nil {
		slog.Error("Failed to setup dependency injection", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := ShutdownDI(injector); err != nil {
			slog.Error("Error during shutdown", "error", err)
		}
	}()

	// Get services from DI container
	cfg := do.MustInvoke[*Config](injector)
	channelMonitor := do.MustInvoke[*ChannelMonitor](injector)
	rssServer := do.MustInvoke[*RSSServer](injector)
	_ = do.MustInvoke[*bot.Bot](injector) // Initialize bot (already done in SetupDI)

	// Start channel monitor
	go channelMonitor.Start(context.Background())

	// Start RSS HTTP server
	go func() {
		if err := rssServer.Start(); err != nil {
			slog.Error("Failed to start RSS server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Bot started", "port", cfg.HTTPPort)
	slog.Info("Press Ctrl+C to stop")

	// Graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	<-ctx.Done()
	slog.Info("Shutting down...")
}
