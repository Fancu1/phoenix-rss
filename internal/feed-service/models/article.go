package models

import "time"

type Article struct {
	ID          uint      `json:"id"`
	FeedID      uint      `json:"feed_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url" gorm:"uniqueIndex"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Read        bool      `json:"read" gorm:"default:false"`
	Starred     bool      `json:"starred" gorm:"default:false"`
	PublishedAt time.Time `json:"published_at"`

	// AI processing fields
	Summary         *string    `json:"summary,omitempty"`
	ProcessingModel *string    `json:"processing_model,omitempty"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
}
