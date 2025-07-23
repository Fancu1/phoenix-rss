package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/Fancu1/phoenix-rss/internal/models"
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

// Legacy methods without context for backward compatibility
// These methods are deprecated and should be migrated to context-aware versions

func (r *ArticleRepository) CreateLegacy(article *models.Article) (*models.Article, error) {
	return r.Create(context.Background(), article)
}

func (r *ArticleRepository) CreateBatchLegacy(articles []*models.Article) error {
	return r.CreateBatch(context.Background(), articles)
}

func (r *ArticleRepository) GetByIDLegacy(id uint) (*models.Article, error) {
	return r.GetByID(context.Background(), id)
}

func (r *ArticleRepository) GetByFeedIDLegacy(feedID uint) ([]*models.Article, error) {
	return r.GetByFeedID(context.Background(), feedID)
}

func (r *ArticleRepository) GetByURLLegacy(url string) (*models.Article, error) {
	return r.GetByURL(context.Background(), url)
}

func (r *ArticleRepository) UpdateLegacy(article *models.Article) (*models.Article, error) {
	return r.Update(context.Background(), article)
}

func (r *ArticleRepository) DeleteLegacy(id uint) error {
	return r.Delete(context.Background(), id)
}

func (r *ArticleRepository) ExistsByURLLegacy(url string) (bool, error) {
	return r.ExistsByURL(context.Background(), url)
}
