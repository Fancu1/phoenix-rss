package models

// Feed represent a simplified feed model for the scheduler service
type Feed struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}
