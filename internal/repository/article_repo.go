package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

func (r *ArticleRepository) Create(article *models.Article) (*models.Article, error) {
	result := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "url"}},
		DoNothing: true,
	}).Create(article)

	return article, result.Error
}

func (r *ArticleRepository) ListByFeedID(feedID uint) ([]*models.Article, error) {
	articles := []*models.Article{}
	result := r.db.Where("feed_id = ?", feedID).Find(&articles)
	return articles, result.Error
}
