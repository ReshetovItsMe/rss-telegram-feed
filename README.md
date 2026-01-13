# RSS Telegram Feed Bot

A Go application that collects RSS feeds from Telegram channel messages and serves them via HTTP.

## Features

- ✅ Connect to Telegram API to read messages from channels
- ✅ Channel selection via chatbot commands
- ✅ Automatic RSS feed generation from channel messages
- ✅ Real-time RSS feed updates
- ✅ Content filtering by keywords
- ✅ Multimedia support (images, videos, documents, audio)
- ✅ RSS feed compatibility with standard RSS clients
- ✅ Access control for authorized users
- ✅ Logging and monitoring
- ✅ Chatbot interface for configuration
- ✅ Scalable architecture

## Requirements

- Go 1.22 or newer
- Telegram Bot Token (get it from [@BotFather](https://t.me/BotFather))
- The bot must be added as an administrator to the channels you want to monitor

### Optional Development Tools

- **Task** - Task runner for development tasks ([install](https://taskfile.dev/installation/))
- **Tilt** - For local development with hot reload ([install](https://docs.tilt.dev/install.html))

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd rss-telegram-feed
```

2. Install dependencies:
```bash
go mod download
```

## Configuration

The application uses [koanf](https://github.com/knadh/koanf) for configuration management. Configuration can be provided via:
- **Config files** (automatically detected): `config.yaml`, `config.yml`, `config.json`, or `config.toml`
- **Environment variables** (always loaded, override config file values)

The application automatically detects and loads configuration files on startup in this order:
1. `config.yaml` (or `config.yml`)
2. `config.json`
3. `config.toml`

### Production Setup

Set environment variables:
```bash
export TELEGRAM_BOT_TOKEN="your_bot_token_here"
export HTTP_PORT="8080"  # Optional, defaults to 8080
export STORAGE_PATH="./data"  # Optional, defaults to ./data
export UPDATE_INTERVAL="60"  # Optional, defaults to 60 seconds
export ALLOWED_USERS="123456789,987654321"  # Optional, comma-separated user IDs
```

### Local Development Setup

1. Copy one of the example config files:
```bash
# For YAML
cp config.example.yaml config.yaml

# Or for JSON
cp config.example.json config.json

# Or for TOML
cp config.example.toml config.toml
```

2. Edit the config file with your values:

**YAML example:**
```yaml
telegram_bot_token: "your_bot_token_here"
http_port: "8080"
storage_path: "./data"
update_interval: 60
```

**JSON example:**
```json
{
  "telegram_bot_token": "your_bot_token_here",
  "http_port": "8080",
  "storage_path": "./data",
  "update_interval": 60
}
```

**TOML example:**
```toml
telegram_bot_token = "your_bot_token_here"
http_port = "8080"
storage_path = "./data"
update_interval = 60
```

3. Run the application:
```bash
go run .
```

The application will automatically detect and load your config file.

### Configuration Options

- `TELEGRAM_BOT_TOKEN` (required): Your Telegram bot token
- `TELEGRAM_API_URL` (optional): Telegram API URL, defaults to `https://api.telegram.org`
- `HTTP_PORT` (optional): Port for RSS HTTP server, defaults to `8080`
- `STORAGE_PATH` (optional): Path for data storage, defaults to `./data`
- `UPDATE_INTERVAL` (optional): Update interval in seconds, defaults to `60`
- `ALLOWED_USERS` (optional): Comma-separated list of allowed user IDs (or array in config files)
- `APP_ENV` (optional): Application environment, defaults to `production`

**Note:** 
- Environment variables always take precedence over config file values
- Config files are automatically detected on application startup
- Supported formats: YAML (`.yaml`, `.yml`), JSON (`.json`), TOML (`.toml`)

## Usage

### Telegram Bot Commands

Once the bot is running, interact with it on Telegram:

- `/start` - Start the bot and see welcome message
- `/help` - Show help message
- `/addchannel @channel_username` - Add a channel to monitor
- `/removechannel <channel_id>` - Remove a channel
- `/listchannels` - List all monitored channels
- `/addfilter <channel_id> <keyword1,keyword2>` - Add keyword filter to a channel
- `/removefilter <channel_id> <filter_index>` - Remove a filter from a channel
- `/rsslink <channel_id>` - Get RSS feed link for a channel (or list all if no ID provided)
- `/status` - Show bot status

### Example Workflow

1. Start the bot: `/start`
2. Add a channel: `/addchannel @example_channel`
3. (Optional) Add filters: `/addfilter 123456789 tech,programming`
4. Get RSS link: `/rsslink 123456789`
5. Use the RSS link in your RSS reader

### RSS Feed Access

RSS feeds are available at:
```
http://localhost:8080/rss/{channel_id}
```

Replace `{channel_id}` with the actual channel ID (shown when you add a channel).

## Architecture

### Components

- **Bot Handler**: Handles Telegram bot commands and user interactions
- **Channel Monitor**: Periodically fetches messages from monitored channels
- **Feed Service**: Generates RSS feeds from stored messages
- **RSS Server**: HTTP server that serves RSS feeds (using Go 1.22 ServeMux)
- **Storage**: File-based storage for channels, messages, and users

### Data Storage

Data is stored in JSON files under the `STORAGE_PATH` directory:
- `channels/` - Channel configurations
- `messages/` - Stored messages organized by channel
- `users/` - Authorized users

## Content Filtering

You can add filters to channels to include or exclude messages based on keywords:

- **Keywords filter**: Only include messages containing at least one of the specified keywords
- **Exclude keywords filter**: Exclude messages containing any of the specified keywords

Example:
```
/addfilter 123456789 tech,programming
```

This will only include messages that contain "tech" or "programming" in their text.

## Multimedia Support

The RSS feed includes information about multimedia attachments:
- Photos
- Videos
- Documents
- Audio files

Media file IDs are included in the RSS feed description. To download media, you would need to use the Telegram Bot API `getFile` method.

## Scaling

The application is designed to be scalable:

- **Horizontal scaling**: Multiple instances can run with shared storage (consider using a database instead of file storage for production)
- **Concurrent processing**: Channel monitoring uses goroutines for parallel processing
- **Stateless HTTP server**: RSS feed generation is stateless and can be load-balanced

For production deployment, consider:
- Using a database (PostgreSQL, MongoDB) instead of file storage
- Implementing Redis for caching
- Using message queues for channel updates
- Setting up proper monitoring and alerting

## Security

- Access control: Only authorized users can configure the bot
- First user auto-authorization: The first user to interact with the bot is automatically authorized (if no `ALLOWED_USERS` is set)
- Input validation: All user inputs are validated
- Error handling: Proper error handling prevents information leakage

## Limitations

- **Channel History**: The Telegram Bot API doesn't provide direct access to channel history. The bot can only receive new messages after it's added to the channel. For existing messages, you would need to use the Telegram Client API (MTProto) which requires different authentication.

- **Message Fetching**: Currently uses `GetUpdates` which may not be the most efficient for production. Consider using webhooks for better performance.

## Development

### Using Taskfile

The project includes a `Taskfile.yml` for common development tasks:

```bash
# Show all available tasks
task

# Install dependencies
task deps

# Build the application
task build

# Run the application
task run

# Run with local config.yaml
task run:local

# Run tests
task test

# Run tests with coverage
task test:coverage

# Format code
task fmt

# Run linter
task lint

# Initial setup (creates config.yaml from template)
task setup

# Clean build artifacts
task clean
```

### Using Tilt

Tilt provides hot reload for local development:

1. Start Tilt:
```bash
tilt up
```

2. Tilt will:
   - Watch for file changes
   - Automatically rebuild and restart the application
   - Show logs in the Tilt UI (usually at http://localhost:10350)

3. Stop Tilt:
```bash
tilt down
```

Or use Taskfile:
```bash
task tilt:up    # Start Tilt
task tilt:down  # Stop Tilt
task tilt:logs  # Show logs
task dev        # Start development environment
```

### Project Structure

```
.
├── main.go              # Application entry point
├── config.go            # Configuration management (using cleanenv)
├── models.go            # Data models
├── storage.go           # Storage interface and file-based implementation
├── bot_handler.go       # Telegram bot command handlers
├── channel_monitor.go   # Channel monitoring service
├── feed_service.go      # RSS feed generation
├── rss_server.go        # HTTP server for RSS feeds
├── di.go                # Dependency injection setup
├── errors.go            # Error definitions
├── Taskfile.yml         # Task runner configuration
├── Tiltfile             # Tilt configuration for local development
├── config.example.yaml  # Example configuration file template
├── go.mod               # Go module file
└── README.md            # This file
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
