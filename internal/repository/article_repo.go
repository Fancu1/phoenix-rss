package repository

import (
	"sync"
	"time"

	"github.com/Fancu1/phoenix-rss/internal/models"
)

type ArticleRepository struct {
	mu       sync.RWMutex
	articles map[uint]*models.Article
	counter  uint
}

func NewArticleRepository() *ArticleRepository {
	return &ArticleRepository{
		articles: make(map[uint]*models.Article),
		counter:  0,
	}
}

func (r *ArticleRepository) Create(article *models.Article) (*models.Article, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// if article already exists, update it
	for _, existingArticle := range r.articles {
		if existingArticle.URL == article.URL {
			existingArticle.Title = article.Title
			existingArticle.Description = article.Description
			existingArticle.Content = article.Content
			existingArticle.UpdatedAt = time.Now()
			return existingArticle, nil
		}
	}

	// create new article
	r.counter++
	article.ID = r.counter
	article.CreatedAt = time.Now()
	article.UpdatedAt = time.Now()
	r.articles[article.ID] = article

	return article, nil
}

func (r *ArticleRepository) ListByFeedID(feedID uint) ([]*models.Article, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	articles := []*models.Article{}
	for _, article := range r.articles {
		if article.FeedID == feedID {
			articles = append(articles, article)
		}
	}

	return articles, nil
}
