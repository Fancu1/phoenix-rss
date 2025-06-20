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
