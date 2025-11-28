package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

type ArticleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

func (r *ArticleRepository) ListByFeedID(ctx context.Context, feedID uint) ([]*models.Article, error) {
	var articles []*models.Article
	err := r.db.WithContext(ctx).
		Where("feed_id = ?", feedID).
		Order("published_at DESC").
		Find(&articles).Error
	return articles, err
}

func (r *ArticleRepository) GetByID(ctx context.Context, articleID uint) (*models.Article, error) {
	var article models.Article
	err := r.db.WithContext(ctx).
		Where("id = ?", articleID).
		First(&article).Error
	if err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *ArticleRepository) GetFeedID(ctx context.Context, articleID uint) (uint, error) {
	var feedID uint
	err := r.db.WithContext(ctx).
		Model(&models.Article{}).
		Select("feed_id").
		Where("id = ?", articleID).
		Scan(&feedID).Error
	return feedID, err
}


