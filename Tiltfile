# Tiltfile for local development
# Tilt will watch for file changes and automatically rebuild/restart the application

# Load environment variables from .env if it exists
load_dotenv()

# Define the local resource
local_resource(
    'rss-telegram-feed',
    cmd='APP_ENV=local go run .',
    deps=[
        'main.go',
        'config.go',
        'bot_handler.go',
        'channel_monitor.go',
        'feed_service.go',
        'rss_server.go',
        'storage.go',
        'models.go',
        'errors.go',
        'di.go',
        'go.mod',
        'go.sum',
        'config.yaml',
    ],
    env={
        'APP_ENV': 'local',
    },
    labels=['rss-telegram-feed'],
    serve_cmd='echo "Application running on http://localhost:8080"',
)

# Watch for changes in all Go files and config
watch_file('*.go')
watch_file('go.mod')
watch_file('go.sum')
watch_file('config.yaml')
