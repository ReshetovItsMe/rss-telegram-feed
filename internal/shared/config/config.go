package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/domain"
	"github.com/reshetovitsme/rss-telegram-feed/internal/shared/errors"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

type Config struct {
	TelegramBotToken string              `koanf:"telegram_bot_token"`
	TelegramAPIURL   string              `koanf:"telegram_api_url"`
	StoragePath      string              `koanf:"storage_path"`
	HTTPPort         string              `koanf:"http_port"`
	UpdateInterval   int                 `koanf:"update_interval"`
	AllowedUsers     []int64             `koanf:"allowed_users"`
	AppEnv           domain.AppEnv       `koanf:"app_env"`
}

func Load() (*Config, error) {
	k := koanf.New(".")

	// Try to load config file from various formats
	configFiles := []string{
		"config.yaml",
		"config.yml",
		"config.json",
		"config.toml",
	}

	// Use lo.Find to find the first existing config file
	configFile, found := lo.Find(configFiles, func(file string) bool {
		_, err := os.Stat(file)
		return err == nil
	})

	if found {
		var parser koanf.Parser
		ext := filepath.Ext(configFile)

		switch ext {
		case ".yaml", ".yml":
			parser = yaml.Parser()
		case ".json":
			parser = json.Parser()
		case ".toml":
			parser = toml.Parser()
		default:
			return nil, oops.Errorf("unsupported config file extension: %s", ext)
		}

		if err := k.Load(file.Provider(configFile), parser); err != nil {
			return nil, oops.With("config_file", configFile).Wrap(err)
		}
	}

	// Load environment variables (they override config file values)
	if err := k.Load(env.Provider("", ".", func(s string) string {
		return strings.ToLower(s)
	}), nil); err != nil {
		return nil, oops.With("context", "loading environment variables").Wrap(err)
	}

	// Set defaults
	if !k.Exists("telegram_api_url") {
		k.Set("telegram_api_url", "https://api.telegram.org")
	}
	if !k.Exists("storage_path") {
		k.Set("storage_path", "./data")
	}
	if !k.Exists("http_port") {
		k.Set("http_port", "8080")
	}
	if !k.Exists("update_interval") {
		k.Set("update_interval", 60)
	}
	if !k.Exists("app_env") {
		k.Set("app_env", "production")
	}

	// Unmarshal into struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, oops.With("context", "unmarshaling config").Wrap(err)
	}

	// Parse AllowedUsers from comma-separated string if it's a string
	if allowedUsers := k.Get("allowed_users"); allowedUsers != nil {
		switch v := allowedUsers.(type) {
		case string:
			cfg.AllowedUsers = ParseAllowedUsers(v)
		case []interface{}:
			cfg.AllowedUsers = lo.FilterMap(v, func(item interface{}, _ int) (int64, bool) {
				switch val := item.(type) {
				case int64:
					return val, true
				case int:
					return int64(val), true
				case float64:
					return int64(val), true
				default:
					return 0, false
				}
			})
		}
	}

	// Parse AppEnv from string if needed
	if appEnvStr := k.String("app_env"); appEnvStr != "" {
		if env, err := domain.ParseAppEnv(appEnvStr); err == nil {
			cfg.AppEnv = env
		} else {
			cfg.AppEnv = domain.AppEnvProduction
		}
	} else {
		cfg.AppEnv = domain.AppEnvProduction
	}

	// Validate required fields
	if cfg.TelegramBotToken == "" {
		return nil, errors.ErrMissingBotToken
	}

	return &cfg, nil
}

// ParseAllowedUsers parses comma-separated user IDs string into []int64
func ParseAllowedUsers(s string) []int64 {
	if s == "" {
		return []int64{}
	}
	parts := strings.Split(s, ",")
	return lo.FilterMap(parts, func(part string, _ int) (int64, bool) {
		part = strings.TrimSpace(part)
		if part == "" {
			return 0, false
		}
		var id int64
		if _, err := fmt.Sscanf(part, "%d", &id); err == nil {
			return id, true
		}
		return 0, false
	})
}
