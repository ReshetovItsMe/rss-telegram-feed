package main

import "errors"

var (
	ErrMissingBotToken = errors.New("TELEGRAM_BOT_TOKEN environment variable is required")
	ErrUnauthorized    = errors.New("unauthorized user")
	ErrChannelNotFound = errors.New("channel not found")
	ErrInvalidFilter   = errors.New("invalid filter")
)
