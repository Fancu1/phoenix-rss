package models

import "time"

type FeedStatus string

const (
	FeedStatusActive FeedStatus = "active"
	FeedStatusError  FeedStatus = "error"
)

type Feed struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	Status      FeedStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
