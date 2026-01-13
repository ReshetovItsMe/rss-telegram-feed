package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	channelDomain "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/domain"
	channelService "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/service"
	feedService "github.com/reshetovitsme/rss-telegram-feed/internal/modules/feed/service"
	messageDomain "github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/domain"
	userDomain "github.com/reshetovitsme/rss-telegram-feed/internal/modules/user/domain"
	userService "github.com/reshetovitsme/rss-telegram-feed/internal/modules/user/service"
	"github.com/reshetovitsme/rss-telegram-feed/internal/shared/config"
)

// Handler handles Telegram bot interactions
type Handler struct {
	cfg            *config.Config
	channelService *channelService.Service
	feedService    *feedService.Service
	userService    *userService.Service
}

// New creates a new Telegram handler
func New(cfg *config.Config, channelService *channelService.Service, feedService *feedService.Service, userService *userService.Service) *Handler {
	return &Handler{
		cfg:            cfg,
		channelService: channelService,
		feedService:    feedService,
		userService:    userService,
	}
}

// RegisterCommands registers bot commands
func (h *Handler) RegisterCommands(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, h.handleStart)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, h.handleHelp)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/addchannel", bot.MatchTypePrefix, h.handleAddChannel)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/removechannel", bot.MatchTypePrefix, h.handleRemoveChannel)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/listchannels", bot.MatchTypeExact, h.handleListChannels)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/addfilter", bot.MatchTypePrefix, h.handleAddFilter)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/removefilter", bot.MatchTypePrefix, h.handleRemoveFilter)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/rsslink", bot.MatchTypePrefix, h.handleRSSLink)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/status", bot.MatchTypeExact, h.handleStatus)
}

// HandleUpdate processes incoming updates
func (h *Handler) HandleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	// Process channel posts and messages
	if update.ChannelPost != nil {
		h.processChannelPost(ctx, b, update.ChannelPost)
	} else if update.Message != nil {
		if update.Message.Chat.Type == "channel" {
			h.processChannelPost(ctx, b, update.Message)
		}
	}
}

func (h *Handler) processChannelPost(ctx context.Context, b *bot.Bot, msg *models.Message) {
	if msg == nil {
		return
	}

	channelID := fmt.Sprintf("%d", msg.Chat.ID)

	// Check if this channel is being monitored
	channel, err := h.channelService.GetChannel(channelID)
	if err != nil {
		// Channel not in our list, ignore
		return
	}

	if !channel.IsActive {
		return
	}

	// Extract message data
	text := msg.Text
	if text == "" && msg.Caption != "" {
		text = msg.Caption
	}

	media := extractMedia(msg)
	author := getAuthorName(msg)
	link := fmt.Sprintf("https://t.me/%s/%d", channel.Username, msg.ID)

	// Process message through channel service
	if err := h.channelService.ProcessMessage(channel, text, int64(msg.Date), int64(msg.ID), author, media, link); err != nil {
		slog.Error("Error processing message", "error", err, "channel_id", channelID, "message_id", msg.ID)
		return
	}

	slog.Info("New message from channel", "channel", channel.Username, "channel_id", channelID, "message_id", msg.ID)
}

func (h *Handler) checkAuthorization(userID int64) bool {
	return h.userService.IsAuthorized(userID, h.cfg.AllowedUsers)
}

func (h *Handler) handleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.Message.From.ID

	if !h.checkAuthorization(userID) {
		// Auto-add first user as admin if no users configured
		if len(h.cfg.AllowedUsers) == 0 {
			user := &userDomain.User{
				ID:       userID,
				Username: update.Message.From.Username,
				AddedAt:  time.Now(),
				IsAdmin:  true,
			}
			if err := h.userService.SaveUser(user); err != nil {
				slog.Error("Failed to save user", "error", err, "user_id", userID)
			}
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❌ You are not authorized to use this bot.",
			})
			return
		}
	}

	text := `👋 Welcome to RSS Telegram Feed Bot!

I help you create RSS feeds from Telegram channels.

Available commands:
/help - Show this help message
/addchannel <channel_username> - Add a channel to monitor
/removechannel <channel_id> - Remove a channel
/listchannels - List all monitored channels
/addfilter <channel_id> <keyword1,keyword2> - Add keyword filter
/removefilter <channel_id> <filter_index> - Remove a filter
/rsslink <channel_id> - Get RSS feed link
/status - Show bot status

Example:
/addchannel @example_channel`

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}

func (h *Handler) handleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.handleStart(ctx, b, update)
}

func (h *Handler) handleAddChannel(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.checkAuthorization(update.Message.From.ID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Unauthorized",
		})
		return
	}

	parts := strings.Fields(update.Message.Text)
	if len(parts) < 2 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Usage: /addchannel <channel_username>\nExample: /addchannel @example_channel",
		})
		return
	}

	channelUsername := strings.TrimPrefix(parts[1], "@")

	// Try to get channel info from Telegram
	chat, err := b.GetChat(ctx, &bot.GetChatParams{
		ChatID: channelUsername,
	})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Failed to get channel info: %v\nMake sure the bot is added to the channel as an administrator.", err),
		})
		return
	}

	channel := &channelDomain.Channel{
		ID:         fmt.Sprintf("%d", chat.ID),
		Username:   channelUsername,
		Title:      chat.Title,
		AddedBy:    update.Message.From.ID,
		AddedAt:    time.Now(),
		Filters:    []channelDomain.Filter{},
		LastUpdate: time.Now(),
		IsActive:   true,
	}

	if err := h.channelService.SaveChannel(channel); err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Failed to save channel: %v", err),
		})
		return
	}

	// Start monitoring this channel
	h.channelService.AddChannel(channel.ID)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("✅ Channel @%s added successfully!\nChannel ID: %s", channelUsername, channel.ID),
	})
}

// Helper functions
func getAuthorName(msg *models.Message) string {
	if msg.From != nil {
		if msg.From.Username != "" {
			return "@" + msg.From.Username
		}
		if msg.From.FirstName != "" {
			return msg.From.FirstName
		}
	}
	return "Unknown"
}

func extractMedia(msg *models.Message) []messageDomain.Media {
	var media []messageDomain.Media

	if msg.Photo != nil && len(msg.Photo) > 0 {
		photo := msg.Photo[len(msg.Photo)-1]
		media = append(media, messageDomain.Media{
			Type:   messageDomain.MediaTypePhoto,
			FileID: photo.FileID,
		})
	}

	if msg.Video != nil {
		media = append(media, messageDomain.Media{
			Type:   messageDomain.MediaTypeVideo,
			FileID: msg.Video.FileID,
		})
	}

	if msg.Document != nil {
		media = append(media, messageDomain.Media{
			Type:   messageDomain.MediaTypeDocument,
			FileID: msg.Document.FileID,
		})
	}

	if msg.Audio != nil {
		media = append(media, messageDomain.Media{
			Type:   messageDomain.MediaTypeAudio,
			FileID: msg.Audio.FileID,
		})
	}

	return media
}

func (h *Handler) handleRemoveChannel(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.checkAuthorization(update.Message.From.ID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Unauthorized",
		})
		return
	}

	parts := strings.Fields(update.Message.Text)
	if len(parts) < 2 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Usage: /removechannel <channel_id>",
		})
		return
	}

	channelID := parts[1]
	if err := h.channelService.DeleteChannel(channelID); err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Failed to remove channel: %v", err),
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("✅ Channel %s removed successfully!", channelID),
	})
}

func (h *Handler) handleListChannels(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.checkAuthorization(update.Message.From.ID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Unauthorized",
		})
		return
	}

	channels, err := h.channelService.GetAllChannels()
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Failed to list channels: %v", err),
		})
		return
	}

	if len(channels) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "📭 No channels added yet.\nUse /addchannel to add one.",
		})
		return
	}

	var text strings.Builder
	text.WriteString("📋 Monitored Channels:\n\n")
	for i, ch := range channels {
		status := "✅"
		if !ch.IsActive {
			status = "⏸️"
		}
		text.WriteString(fmt.Sprintf("%s %d. @%s\n   ID: %s\n   Filters: %d\n\n",
			status, i+1, ch.Username, ch.ID, len(ch.Filters)))
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text.String(),
	})
}

func (h *Handler) handleAddFilter(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.checkAuthorization(update.Message.From.ID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Unauthorized",
		})
		return
	}

	parts := strings.Fields(update.Message.Text)
	if len(parts) < 3 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Usage: /addfilter <channel_id> <keyword1,keyword2,...>\nExample: /addfilter 123456789 tech,programming",
		})
		return
	}

	channelID := parts[1]
	keywords := strings.Split(parts[2], ",")

	channel, err := h.channelService.GetChannel(channelID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Channel not found: %s", channelID),
		})
		return
	}

	filter := channelDomain.Filter{
		Type:     channelDomain.FilterTypeKeywords,
		Keywords: keywords,
		Enabled:  true,
	}

	channel.Filters = append(channel.Filters, filter)

	if err := h.channelService.SaveChannel(channel); err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Failed to save filter: %v", err),
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("✅ Filter added to channel %s\nKeywords: %v", channelID, keywords),
	})
}

func (h *Handler) handleRemoveFilter(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.checkAuthorization(update.Message.From.ID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Unauthorized",
		})
		return
	}

	parts := strings.Fields(update.Message.Text)
	if len(parts) < 3 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Usage: /removefilter <channel_id> <filter_index>",
		})
		return
	}

	channelID := parts[1]
	index, err := strconv.Atoi(parts[2])
	if err != nil || index < 1 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Invalid filter index",
		})
		return
	}

	channel, err := h.channelService.GetChannel(channelID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Channel not found: %s", channelID),
		})
		return
	}

	if index > len(channel.Filters) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Filter index out of range",
		})
		return
	}

	channel.Filters = append(channel.Filters[:index-1], channel.Filters[index:]...)

	if err := h.channelService.SaveChannel(channel); err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Failed to remove filter: %v", err),
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("✅ Filter %d removed from channel %s", index, channelID),
	})
}

func (h *Handler) handleRSSLink(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.checkAuthorization(update.Message.From.ID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Unauthorized",
		})
		return
	}

	parts := strings.Fields(update.Message.Text)
	channelID := ""
	if len(parts) >= 2 {
		channelID = parts[1]
	}

	if channelID == "" {
		// List all RSS links
		channels, err := h.channelService.GetAllChannels()
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   fmt.Sprintf("❌ Failed to get channels: %v", err),
			})
			return
		}

		var text strings.Builder
		text.WriteString("🔗 RSS Feed Links:\n\n")
		for _, ch := range channels {
			link := fmt.Sprintf("http://localhost:%s/rss/%s", h.cfg.HTTPPort, ch.ID)
			text.WriteString(fmt.Sprintf("@%s:\n%s\n\n", ch.Username, link))
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   text.String(),
		})
		return
	}

	// Get specific channel RSS link
	channel, err := h.channelService.GetChannel(channelID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Channel not found: %s", channelID),
		})
		return
	}

	link := fmt.Sprintf("http://localhost:%s/rss/%s", h.cfg.HTTPPort, channel.ID)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("🔗 RSS Feed for @%s:\n%s", channel.Username, link),
	})
}

func (h *Handler) handleStatus(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.checkAuthorization(update.Message.From.ID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Unauthorized",
		})
		return
	}

	channels, err := h.channelService.GetAllChannels()
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("❌ Failed to get status: %v", err),
		})
		return
	}

	activeCount := 0
	for _, ch := range channels {
		if ch.IsActive {
			activeCount++
		}
	}

	text := fmt.Sprintf(`📊 Bot Status:

Channels: %d (Active: %d)
Update Interval: %d seconds
HTTP Port: %s
Storage: %s`,
		len(channels), activeCount, h.cfg.UpdateInterval, h.cfg.HTTPPort, h.cfg.StoragePath)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}
