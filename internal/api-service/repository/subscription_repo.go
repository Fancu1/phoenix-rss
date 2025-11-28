package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

type SubscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		Count(&count).Error
	return count > 0, err
}

func (r *SubscriptionRepository) ListUserFeeds(ctx context.Context, userID uint) ([]*models.UserFeed, error) {
	var subscriptions []models.Subscription
	err := r.db.WithContext(ctx).
		Preload("Feed").
		Where("user_id = ?", userID).
		Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}

	result := make([]*models.UserFeed, len(subscriptions))
	for i, sub := range subscriptions {
		result[i] = &models.UserFeed{
			Feed:        sub.Feed,
			CustomTitle: sub.CustomTitle,
		}
	}
	return result, nil
}

func (r *SubscriptionRepository) UpdateCustomTitle(ctx context.Context, userID, feedID uint, title *string) error {
	return r.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		Update("custom_title", title).Error
}

func (r *SubscriptionRepository) Delete(ctx context.Context, userID, feedID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		Delete(&models.Subscription{}).Error
}

func (r *SubscriptionRepository) GetWithFeed(ctx context.Context, userID, feedID uint) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.WithContext(ctx).
		Preload("Feed").
		Where("user_id = ? AND feed_id = ?", userID, feedID).
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}


