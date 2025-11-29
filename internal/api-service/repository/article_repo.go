package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

const (
	// DefaultPageSize is the default number of articles per page
	DefaultPageSize = 8
	// MaxPageSize prevents excessive data transfer in a single request
	MaxPageSize = 50
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

// ListByFeedIDPaginated returns paginated articles for a feed.
// Results are ordered by published_at DESC (newest first).
// Page numbers start from 1. Invalid inputs are normalized to defaults.
func (r *ArticleRepository) ListByFeedIDPaginated(
	ctx context.Context,
	feedID uint,
	page, pageSize int,
) ([]*models.Article, int64, error) {
	// Normalize inputs to prevent invalid queries
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		pageSize = DefaultPageSize
	}

	offset := (page - 1) * pageSize

	// Count total articles first (uses idx_articles_feed_id)
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.Article{}).
		Where("feed_id = ?", feedID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch paginated articles (uses idx_articles_feed_published)
	var articles []*models.Article
	if err := r.db.WithContext(ctx).
		Where("feed_id = ?", feedID).
		Order("published_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&articles).Error; err != nil {
		return nil, 0, err
	}

	return articles, total, nil
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


