package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/models"
	"github.com/Fancu1/phoenix-rss/internal/repository"
)

type ArticleServiceInterface interface {
	FetchAndSaveArticles(ctx context.Context, feedID uint) ([]*models.Article, error)
	ListArticlesByFeedID(ctx context.Context, feedID uint) ([]*models.Article, error)
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
	log := logger.FromContext(ctx)

	log.Info("fetching articles for feed", "feed_id", feedID)

	// Get the feed
	feed, err := s.feedRepo.GetByID(ctx, feedID)
	if err != nil {
		log.Error("failed to get feed", "feed_id", feedID, "error", err.Error())
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	log.Info("parsing feed from URL", "feed_id", feedID, "url", feed.URL)

	// Parse the feed
	parsedFeed, err := s.parser.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		log.Error("failed to parse feed", "feed_id", feedID, "url", feed.URL, "error", err.Error())
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	log.Info("parsed feed successfully", "feed_id", feedID, "article_count", len(parsedFeed.Items))

	var articles []*models.Article
	var newArticles []*models.Article

	for _, item := range parsedFeed.Items {
		// Check if article already exists
		exists, err := s.articleRepo.ExistsByURL(ctx, item.Link)
		if err != nil {
			log.Warn("failed to check if article exists", "url", item.Link, "error", err.Error())
			continue
		}

		if exists {
			log.Debug("article already exists, skipping", "url", item.Link)
			continue
		}

		publishedAt := time.Now()
		if item.PublishedParsed != nil {
			publishedAt = *item.PublishedParsed
		}

		article := &models.Article{
			Title:       item.Title,
			URL:         item.Link,
			Description: item.Description,
			Content:     item.Content,
			FeedID:      feedID,
			PublishedAt: publishedAt,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		articles = append(articles, article)
		newArticles = append(newArticles, article)

		log.Debug("prepared new article", "title", item.Title, "url", item.Link)
	}

	if len(newArticles) == 0 {
		log.Info("no new articles to save", "feed_id", feedID)
		return articles, nil
	}

	log.Info("saving new articles", "feed_id", feedID, "new_article_count", len(newArticles))

	// Save all new articles in batch
	err = s.articleRepo.CreateBatch(ctx, newArticles)
	if err != nil {
		log.Error("failed to save articles", "feed_id", feedID, "error", err.Error())
		return nil, fmt.Errorf("failed to save articles: %w", err)
	}

	log.Info("successfully saved articles", "feed_id", feedID, "saved_count", len(newArticles))
	return articles, nil
}

func (s *ArticleService) ListArticlesByFeedID(ctx context.Context, feedID uint) ([]*models.Article, error) {
	log := logger.FromContext(ctx)

	log.Info("listing articles for feed", "feed_id", feedID)

	articles, err := s.articleRepo.GetByFeedID(ctx, feedID)
	if err != nil {
		log.Error("failed to list articles", "feed_id", feedID, "error", err.Error())
		return nil, fmt.Errorf("failed to list articles: %w", err)
	}

	log.Info("successfully listed articles", "feed_id", feedID, "count", len(articles))
	return articles, nil
}
