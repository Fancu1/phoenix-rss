package repository

import (
	"context"

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

// ListByUserID returns feeds subscribed by a specific user
func (r *FeedRepository) ListByUserID(ctx context.Context, userID uint) ([]*models.Feed, error) {
	feeds := make([]*models.Feed, 0)
	result := r.db.WithContext(ctx).
		Joins("JOIN subscriptions ON subscriptions.feed_id = feeds.id").
		Where("subscriptions.user_id = ?", userID).
		Find(&feeds)
	return feeds, result.Error
}

// CreateSubscription creates a new subscription between user and feed
func (r *FeedRepository) CreateSubscription(ctx context.Context, subscription *models.Subscription) error {
	result := r.db.WithContext(ctx).Create(subscription)
	return result.Error
}

// DeleteSubscription removes a subscription between user and feed
func (r *FeedRepository) DeleteSubscription(ctx context.Context, userID, feedID uint) error {
	result := r.db.WithContext(ctx).Where("user_id = ? AND feed_id = ?", userID, feedID).Delete(&models.Subscription{})
	return result.Error
}

// IsUserSubscribed checks if a user is subscribed to a feed
func (r *FeedRepository) IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&models.Subscription{}).
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		Count(&count)
	return count > 0, result.Error
}

// Legacy methods without context for backward compatibility
// These methods are deprecated and should be migrated to context-aware versions

func (r *FeedRepository) CreateLegacy(feed *models.Feed) (*models.Feed, error) {
	return r.Create(context.Background(), feed)
}

func (r *FeedRepository) UpdateLegacy(feed *models.Feed) (*models.Feed, error) {
	return r.Update(context.Background(), feed)
}

func (r *FeedRepository) ListAllLegacy() ([]*models.Feed, error) {
	return r.ListAll(context.Background())
}

func (r *FeedRepository) GetByIDLegacy(id uint) (*models.Feed, error) {
	return r.GetByID(context.Background(), id)
}

func (r *FeedRepository) ListByUserIDLegacy(userID uint) ([]*models.Feed, error) {
	return r.ListByUserID(context.Background(), userID)
}

func (r *FeedRepository) CreateSubscriptionLegacy(subscription *models.Subscription) error {
	return r.CreateSubscription(context.Background(), subscription)
}

func (r *FeedRepository) DeleteSubscriptionLegacy(userID, feedID uint) error {
	return r.DeleteSubscription(context.Background(), userID, feedID)
}

func (r *FeedRepository) IsUserSubscribedLegacy(userID, feedID uint) (bool, error) {
	return r.IsUserSubscribed(context.Background(), userID, feedID)
}
