package models

import "time"

type Subscription struct {
	UserID      uint      `gorm:"primaryKey"`
	FeedID      uint      `gorm:"primaryKey"`
	CustomTitle *string   `json:"custom_title,omitempty" gorm:"size:255"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Associations
	Feed Feed `gorm:"foreignKey:FeedID"`
}
