package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	feedService "github.com/reshetovitsme/rss-telegram-feed/internal/modules/feed/service"
	"github.com/reshetovitsme/rss-telegram-feed/internal/shared/config"
	sloghttp "github.com/samber/slog-http"
)

// Server handles HTTP requests for RSS feeds
type Server struct {
	cfg         *config.Config
	feedService *feedService.Service
	logger      *slog.Logger
}

// New creates a new HTTP server
func New(cfg *config.Config, feedService *feedService.Service) *Server {
	return &Server{
		cfg:         cfg,
		feedService: feedService,
		logger:      slog.Default(),
	}
}

// SetLogger sets the logger
func (s *Server) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// RSS feed endpoint
	mux.HandleFunc("GET /rss/{channelID}", s.handleRSSFeed)

	// Health check endpoint
	mux.HandleFunc("GET /health", s.handleHealth)

	// Root endpoint with instructions
	mux.HandleFunc("GET /", s.handleRoot)

	addr := fmt.Sprintf(":%s", s.cfg.HTTPPort)
	s.logger.Info("RSS server starting", "addr", addr)

	// Use slog-http middleware with recovery
	handler := sloghttp.Recovery(mux)
	handler = sloghttp.New(s.logger)(handler)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

func (s *Server) handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("channelID")
	if channelID == "" {
		http.Error(w, "Channel ID is required", http.StatusBadRequest)
		return
	}

	// Get base URL from request
	baseURL := fmt.Sprintf("%s://%s", getScheme(r), r.Host)

	feed, err := s.feedService.GenerateFeed(channelID, baseURL)
	if err != nil {
		s.logger.Error("Error generating feed", "channel_id", channelID, "error", err)
		http.Error(w, "Failed to generate feed", http.StatusInternalServerError)
		return
	}

	// Generate RSS XML
	rss, err := feed.ToRss()
	if err != nil {
		s.logger.Error("Error converting feed to RSS", "error", err)
		http.Error(w, "Failed to generate RSS", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(rss))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>RSS Telegram Feed</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .info { background: #f5f5f5; padding: 15px; border-radius: 5px; margin: 20px 0; }
        code { background: #e8e8e8; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>RSS Telegram Feed Service</h1>
    <div class="info">
        <p>This service provides RSS feeds from Telegram channels.</p>
        <p>To access a feed, use: <code>/rss/{channelID}</code></p>
        <p>Example: <code>/rss/123456789</code></p>
    </div>
    <p><a href="/health">Health Check</a></p>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}
