package repository

import (
	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/models"
)

type FeedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{
		db: db,
	}
}

func (r *FeedRepository) Create(feed *models.Feed) (*models.Feed, error) {
	result := r.db.Create(feed)
	return feed, result.Error
}

func (r *FeedRepository) Update(feed *models.Feed) (*models.Feed, error) {
	result := r.db.Save(feed)
	return feed, result.Error
}

func (r *FeedRepository) ListAll() ([]*models.Feed, error) {
	feeds := make([]*models.Feed, 0)
	result := r.db.Find(&feeds)
	return feeds, result.Error
}

func (r *FeedRepository) GetByID(id uint) (*models.Feed, error) {
	feed := &models.Feed{}
	result := r.db.First(feed, id)
	return feed, result.Error
}

// ListByUserID returns feeds subscribed by a specific user
func (r *FeedRepository) ListByUserID(userID uint) ([]*models.Feed, error) {
	feeds := make([]*models.Feed, 0)
	result := r.db.
		Joins("JOIN subscriptions ON subscriptions.feed_id = feeds.id").
		Where("subscriptions.user_id = ?", userID).
		Find(&feeds)
	return feeds, result.Error
}

// CreateSubscription creates a new subscription between user and feed
func (r *FeedRepository) CreateSubscription(subscription *models.Subscription) error {
	result := r.db.Create(subscription)
	return result.Error
}

// DeleteSubscription removes a subscription between user and feed
func (r *FeedRepository) DeleteSubscription(userID, feedID uint) error {
	result := r.db.Where("user_id = ? AND feed_id = ?", userID, feedID).Delete(&models.Subscription{})
	return result.Error
}

// IsUserSubscribed checks if a user is subscribed to a feed
func (r *FeedRepository) IsUserSubscribed(userID, feedID uint) (bool, error) {
	var count int64
	result := r.db.Model(&models.Subscription{}).
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		Count(&count)
	return count > 0, result.Error
}
