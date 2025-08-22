package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

type FeedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{
		db: db,
	}
}

func (r *FeedRepository) Create(ctx context.Context, feed *models.Feed) (*models.Feed, error) {
	result := r.db.WithContext(ctx).Create(feed)
	return feed, result.Error
}

func (r *FeedRepository) Update(ctx context.Context, feed *models.Feed) (*models.Feed, error) {
	result := r.db.WithContext(ctx).Save(feed)
	return feed, result.Error
}

func (r *FeedRepository) ListAll(ctx context.Context) ([]*models.Feed, error) {
	feeds := make([]*models.Feed, 0)
	result := r.db.WithContext(ctx).Find(&feeds)
	return feeds, result.Error
}

func (r *FeedRepository) GetByID(ctx context.Context, id uint) (*models.Feed, error) {
	feed := &models.Feed{}
	result := r.db.WithContext(ctx).First(feed, id)
	return feed, result.Error
}

func (r *FeedRepository) GetByURL(ctx context.Context, url string) (*models.Feed, error) {
	feed := &models.Feed{}
	result := r.db.WithContext(ctx).Where("url = ?", url).First(feed)
	if result.Error != nil {
		return nil, result.Error
	}
	return feed, nil
}

func (r *FeedRepository) ListByUserID(ctx context.Context, userID uint) ([]*models.Feed, error) {
	feeds := make([]*models.Feed, 0)
	result := r.db.WithContext(ctx).
		Joins("JOIN subscriptions ON subscriptions.feed_id = feeds.id").
		Where("subscriptions.user_id = ?", userID).
		Find(&feeds)
	return feeds, result.Error
}

func (r *FeedRepository) CreateSubscription(ctx context.Context, subscription *models.Subscription) error {
	result := r.db.WithContext(ctx).Create(subscription)
	return result.Error
}

func (r *FeedRepository) DeleteSubscription(ctx context.Context, userID, feedID uint) error {
	result := r.db.WithContext(ctx).Where("user_id = ? AND feed_id = ?", userID, feedID).Delete(&models.Subscription{})
	return result.Error
}

// IsUserSubscribed check if a user is subscribed to a feed
func (r *FeedRepository) IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&models.Subscription{}).
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		Count(&count)
	return count > 0, result.Error
}
