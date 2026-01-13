//go:generate go run github.com/abice/go-enum --file=$GOFILE --names --nocase

package domain

// FilterType represents the type of content filter
// ENUM(keywords,exclude_keywords,author)
type FilterType string

// AppEnv represents the application environment
// ENUM(local,production,development,testing)
type AppEnv string
