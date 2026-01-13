package main

//go:generate go run github.com/abice/go-enum --file=$GOFILE --names --nocase

// FilterType represents the type of content filter
// ENUM(keywords,exclude_keywords,author)
type FilterType string

// MediaType represents the type of media content
// ENUM(photo,video,document,audio)
type MediaType string
