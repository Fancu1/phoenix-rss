package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Fancu1/phoenix-rss/internal/models"
	"github.com/Fancu1/phoenix-rss/internal/repository"
)

type ArticleServiceInterface interface {
	FetchAndSaveArticles(ctx context.Context, feedID uint) ([]*models.Article, error)
	ListArticlesByFeedID(feedID uint) ([]*models.Article, error)
}

type ArticleService struct {
	parser      *gofeed.Parser
	feedRepo    *repository.FeedRepository
	articleRepo *repository.ArticleRepository
	logger      *slog.Logger
}

func NewArticleService(feedRepo *repository.FeedRepository, articleRepo *repository.ArticleRepository, logger *slog.Logger) *ArticleService {
	return &ArticleService{
		parser:      gofeed.NewParser(),
		feedRepo:    feedRepo,
		articleRepo: articleRepo,
		logger:      logger,
	}
}

func (s *ArticleService) FetchAndSaveArticles(ctx context.Context, feedID uint) ([]*models.Article, error) {
	feed, err := s.feedRepo.GetByID(feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	parsedFeed, err := s.parser.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	if parsedFeed.Title != feed.Title || parsedFeed.Description != feed.Description {
		feed.Title = parsedFeed.Title
		feed.Description = parsedFeed.Description
		_, err := s.feedRepo.Update(feed)
		if err != nil {
			s.logger.Warn("failed to update feed metadata", "feed_id", feedID, "error", err)
		}
	}

	existingArticles, err := s.articleRepo.ListByFeedID(feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}

	existArticles := make(map[string]bool)
	for _, article := range existingArticles {
		existArticles[article.URL] = true
	}

	var articlesToCreate []*models.Article
	for _, item := range parsedFeed.Items {
		if _, ok := existArticles[item.Link]; ok {
			continue
		}

		article := &models.Article{
			FeedID:      feedID,
			Title:       item.Title,
			URL:         item.Link,
			Description: item.Description,
			Content:     item.Content,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Read:        false,
			Starred:     false,
		}
		if item.PublishedParsed != nil {
			article.PublishedAt = *item.PublishedParsed
		}

		articlesToCreate = append(articlesToCreate, article)
	}

	savedArticles, err := s.articleRepo.CreateMany(articlesToCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to save articles: %w", err)
	}

	s.logger.Info("saved articles", "feed_id", feedID, "count", len(savedArticles))

	return savedArticles, nil
}

func (s *ArticleService) ListArticlesByFeedID(feedID uint) ([]*models.Article, error) {
	return s.articleRepo.ListByFeedID(feedID)
}
