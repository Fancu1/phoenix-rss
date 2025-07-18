package models

import "time"

type Subscription struct {
	UserID    uint      `gorm:"primaryKey"`
	FeedID    uint      `gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	User User `gorm:"foreignKey:UserID"`
	Feed Feed `gorm:"foreignKey:FeedID"`
}
