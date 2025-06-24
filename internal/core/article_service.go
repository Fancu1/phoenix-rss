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

	var savedArticles []*models.Article
	for _, item := range parsedFeed.Items {
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

		saved, err := s.articleRepo.Create(article)
		if err != nil {
			s.logger.Error("failed to save article, error: %w", err)
			continue
		}
		savedArticles = append(savedArticles, saved)
	}

	return savedArticles, nil
}

func (s *ArticleService) ListArticlesByFeedID(feedID uint) ([]*models.Article, error) {
	return s.articleRepo.ListByFeedID(feedID)
}
