package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
)

type ArticleRepository struct {
	db *gorm.DB
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
