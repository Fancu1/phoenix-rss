package models

import "time"

type Article struct {
	ID          uint      `json:"id"`
	FeedID      uint      `json:"feed_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Read        bool      `json:"read" gorm:"default:false"`
	Starred     bool      `json:"starred" gorm:"default:false"`
	PublishedAt time.Time `json:"published_at"`
}
