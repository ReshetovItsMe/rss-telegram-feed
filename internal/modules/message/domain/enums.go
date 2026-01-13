//go:generate go run github.com/abice/go-enum --file=$GOFILE --names --nocase

package domain

// MediaType represents the type of media content
// ENUM(photo,video,document,audio)
type MediaType string
