package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

type ArticleRepository struct {
	db *gorm.DB
}

type ArticleCheckCursor struct {
	PublishedAt time.Time
	ArticleID   uint
}

type ArticleCheckCandidate struct {
	ID               uint
	FeedID           uint
	URL              string
	HTTPETag         *string `gorm:"column:http_etag"`
	HTTPLastModified *string `gorm:"column:http_last_modified"`
	PublishedAt      time.Time
}

func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{
		db: db,
	}
}

func (r *ArticleRepository) Create(ctx context.Context, article *models.Article) (*models.Article, error) {
	result := r.db.WithContext(ctx).Create(article)
	return article, result.Error
}

func (r *ArticleRepository) CreateBatch(ctx context.Context, articles []*models.Article) error {
	if len(articles) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).Create(articles)
	return result.Error
}

func (r *ArticleRepository) GetByID(ctx context.Context, id uint) (*models.Article, error) {
	article := &models.Article{}
	result := r.db.WithContext(ctx).First(article, id)
	return article, result.Error
}

func (r *ArticleRepository) GetByFeedID(ctx context.Context, feedID uint) ([]*models.Article, error) {
	articles := make([]*models.Article, 0)
	result := r.db.WithContext(ctx).Where("feed_id = ?", feedID).Order("published_at DESC").Find(&articles)
	return articles, result.Error
}

func (r *ArticleRepository) GetByURL(ctx context.Context, url string) (*models.Article, error) {
	article := &models.Article{}
	result := r.db.WithContext(ctx).Where("url = ?", url).First(article)
	return article, result.Error
}

func (r *ArticleRepository) Update(ctx context.Context, article *models.Article) (*models.Article, error) {
	result := r.db.WithContext(ctx).Save(article)
	return article, result.Error
}

func (r *ArticleRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Article{}, id)
	return result.Error
}

func (r *ArticleRepository) ExistsByURL(ctx context.Context, url string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&models.Article{}).Where("url = ?", url).Count(&count)
	return count > 0, result.Error
}

func (r *ArticleRepository) UpdateWithAIData(ctx context.Context, articleID uint, summary string, processingModel string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
		"summary":          summary,
		"processing_model": processingModel,
		"processed_at":     now,
	})
	return result.Error
}

func (r *ArticleRepository) ListArticlesToCheck(
	ctx context.Context,
	publishedSince, lastCheckedBefore time.Time,
	limit int,
	cursor *ArticleCheckCursor,
) ([]ArticleCheckCandidate, *ArticleCheckCursor, error) {
	if limit <= 0 {
		return nil, nil, fmt.Errorf("limit must be greater than zero")
	}

	query := r.db.WithContext(ctx).
		Model(&models.Article{}).
		Select("id, feed_id, url, http_etag, http_last_modified, published_at").
		Where("published_at >= ?", publishedSince).
		Where("last_checked_at IS NULL OR last_checked_at <= ?", lastCheckedBefore)

	if cursor != nil {
		query = query.Where("(published_at < ?) OR (published_at = ? AND id > ?)", cursor.PublishedAt, cursor.PublishedAt, cursor.ArticleID)
	}

	var records []ArticleCheckCandidate
	if err := query.Order("published_at DESC, id ASC").Limit(limit).Find(&records).Error; err != nil {
		return nil, nil, err
	}

	if len(records) == 0 {
		return records, nil, nil
	}

	last := records[len(records)-1]
	return records, &ArticleCheckCursor{PublishedAt: last.PublishedAt, ArticleID: last.ID}, nil
}

func (r *ArticleRepository) MarkLastChecked(ctx context.Context, articleID uint, checkedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&models.Article{}).
		Where("id = ?", articleID).
		Update("last_checked_at", checkedAt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("article %d not found: %w", articleID, gorm.ErrRecordNotFound)
	}
	return nil
}

func (r *ArticleRepository) UpdateArticleOnChange(
	ctx context.Context,
	articleID uint,
	content, description string,
	newETag, newLastModified *string,
	checkedAt time.Time,
	prevETag, prevLastModified *string,
) (bool, error) {
	updates := map[string]interface{}{
		"content":            content,
		"description":        description,
		"last_checked_at":    checkedAt,
		"updated_at":         checkedAt,
		"http_etag":          newETag,
		"http_last_modified": newLastModified,
	}

	query := r.db.WithContext(ctx).Model(&models.Article{}).Where("id = ?", articleID)

	if prevETag != nil {
		query = query.Where("http_etag = ?", *prevETag)
	} else {
		query = query.Where("http_etag IS NULL")
	}

	if prevLastModified != nil {
		query = query.Where("http_last_modified = ?", *prevLastModified)
	} else {
		query = query.Where("http_last_modified IS NULL")
	}

	result := query.Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}

	if result.RowsAffected == 0 {
		return false, nil
	}

	return true, nil
}
