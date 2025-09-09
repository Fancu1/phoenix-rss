package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
	article_eventspb "github.com/Fancu1/phoenix-rss/proto/gen/article_events"
)

type ArticleServiceInterface interface {
	FetchAndSaveArticles(ctx context.Context, feedID uint) ([]*models.Article, error)
	ListArticlesByFeedID(ctx context.Context, userID, feedID uint) ([]*models.Article, error)
	HandleArticleProcessed(ctx context.Context, event *article_eventspb.ArticleProcessedEvent) error
}

type ArticleService struct {
	parser        *gofeed.Parser
	feedRepo      *repository.FeedRepository
	articleRepo   *repository.ArticleRepository
	eventProducer events.ArticleEventProducer
	logger        *slog.Logger
}

func NewArticleService(feedRepo *repository.FeedRepository, articleRepo *repository.ArticleRepository, eventProducer events.ArticleEventProducer, logger *slog.Logger) *ArticleService {
	return &ArticleService{
		parser:        gofeed.NewParser(),
		feedRepo:      feedRepo,
		articleRepo:   articleRepo,
		eventProducer: eventProducer,
		logger:        logger,
	}
}

func (s *ArticleService) FetchAndSaveArticles(ctx context.Context, feedID uint) ([]*models.Article, error) {
	log := logger.FromContext(ctx)

	log.Info("fetching articles for feed", "feed_id", feedID)

	feed, err := s.feedRepo.GetByID(ctx, feedID)
	if err != nil {
		log.Error("failed to get feed", "feed_id", feedID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to get feed %d for article fetch: %w", feedID, err))
	}

	if feed == nil {
		log.Error("feed not found", "feed_id", feedID)
		return nil, fmt.Errorf("feed %d not found: %w", feedID, ierr.ErrFeedNotFound)
	}

	log.Info("parsing feed from URL", "feed_id", feedID, "url", feed.URL)

	parsedFeed, err := s.parser.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		log.Error("failed to parse feed", "feed_id", feedID, "url", feed.URL, "error", err.Error())
		return nil, fmt.Errorf("failed to parse feed %d (%s) from URL '%s': %w", feedID, feed.Title, feed.URL, ierr.ErrFeedFetchFailed.WithCause(err))
	}

	log.Info("parsed feed successfully", "feed_id", feedID, "article_count", len(parsedFeed.Items))

	var articles []*models.Article
	var newArticles []*models.Article

	for _, item := range parsedFeed.Items {
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

	err = s.articleRepo.CreateBatch(ctx, newArticles)
	if err != nil {
		log.Error("failed to save articles", "feed_id", feedID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to save %d articles for feed %d (%s): %w", len(newArticles), feedID, feed.Title, err))
	}

	log.Info("successfully saved articles", "feed_id", feedID, "saved_count", len(newArticles))

	// Publish ArticlePersistedEvent for each new article
	if s.eventProducer != nil {
		for _, article := range newArticles {
			event := &article_eventspb.ArticlePersistedEvent{
				ArticleId:   uint64(article.ID),
				FeedId:      uint64(article.FeedID),
				Title:       article.Title,
				Content:     article.Content,
				Url:         article.URL,
				Description: article.Description,
				PublishedAt: article.PublishedAt.Unix(),
			}

			if err := s.eventProducer.PublishArticlePersisted(ctx, event); err != nil {
				log.Error("failed to publish article persisted event",
					"article_id", article.ID,
					"feed_id", feedID,
					"error", err.Error())
			} else {
				log.Debug("published article persisted event",
					"article_id", article.ID,
					"feed_id", feedID)
			}
		}
	}

	return articles, nil
}

func (s *ArticleService) ListArticlesByFeedID(ctx context.Context, userID, feedID uint) ([]*models.Article, error) {
	log := logger.FromContext(ctx)

	log.Info("listing articles for feed", "user_id", userID, "feed_id", feedID)

	isSubscribed, err := s.feedRepo.IsUserSubscribed(ctx, userID, feedID)
	if err != nil {
		log.Error("failed to check subscription", "user_id", userID, "feed_id", feedID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to check subscription for user %d and feed %d: %w", userID, feedID, err))
	}

	if !isSubscribed {
		log.Warn("user not subscribed to feed", "user_id", userID, "feed_id", feedID)
		return nil, ierr.ErrNotSubscribed
	}

	articles, err := s.articleRepo.GetByFeedID(ctx, feedID)
	if err != nil {
		log.Error("failed to list articles", "feed_id", feedID, "error", err.Error())
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to list articles for feed %d: %w", feedID, err))
	}

	log.Info("successfully listed articles", "user_id", userID, "feed_id", feedID, "count", len(articles))
	return articles, nil
}

// HandleArticleProcessed handles an ArticleProcessedEvent by updating the article with AI data
func (s *ArticleService) HandleArticleProcessed(ctx context.Context, event *article_eventspb.ArticleProcessedEvent) error {
	log := logger.FromContext(ctx)

	log.Info("handling article processed event",
		"article_id", event.ArticleId,
		"summary_length", len(event.Summary),
		"processing_model", event.ProcessingModel,
	)

	// Validate the event
	if event.ArticleId == 0 {
		return fmt.Errorf("invalid article ID in processed event: %d", event.ArticleId)
	}

	// Update the article with AI data
	err := s.articleRepo.UpdateWithAIData(
		ctx,
		uint(event.ArticleId),
		event.Summary,
		event.ProcessingModel,
	)
	if err != nil {
		log.Error("failed to update article with AI data",
			"article_id", event.ArticleId,
			"error", err.Error())
		return ierr.NewDatabaseError(fmt.Errorf("failed to update article %d with AI data: %w", event.ArticleId, err))
	}

	log.Info("successfully updated article with AI data",
		"article_id", event.ArticleId,
		"summary_length", len(event.Summary),
	)

	return nil
}
