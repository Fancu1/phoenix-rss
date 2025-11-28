package worker

import (
	"context"
	"log/slog"

	"github.com/mmcdole/gofeed"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/core"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

// FeedFetcher consumes events and triggers article fetching
type FeedFetcher struct {
	logger         *slog.Logger
	articleService *core.ArticleService
	feedRepo       *repository.FeedRepository
	parser         *gofeed.Parser
}

func NewFeedFetcher(logger *slog.Logger, articleService *core.ArticleService, feedRepo *repository.FeedRepository) *FeedFetcher {
	return &FeedFetcher{
		logger:         logger,
		articleService: articleService,
		feedRepo:       feedRepo,
		parser:         gofeed.NewParser(),
	}
}

// HandleFeedFetch fetches articles and updates feed metadata if needed.
func (f *FeedFetcher) HandleFeedFetch(ctx context.Context, evt events.FeedFetchEvent) error {
	taskCtx := logger.WithValue(ctx, "feed_id", evt.FeedID)
	log := logger.FromContext(taskCtx)
	log.Info("starting feed fetch", "feed_id", evt.FeedID)

	feed, err := f.feedRepo.GetByID(ctx, evt.FeedID)
	if err != nil {
		log.Error("failed to get feed", "feed_id", evt.FeedID, "error", err.Error())
		return err
	}

	needsMetadataUpdate := feed.Title == feed.URL // title == URL means first fetch

	articles, err := f.articleService.FetchAndSaveArticles(taskCtx, evt.FeedID)
	if err != nil {
		log.Error("failed to fetch and save articles for feed", "feed_id", evt.FeedID, "error", err.Error())
		if updateErr := f.feedRepo.UpdateStatus(ctx, evt.FeedID, models.FeedStatusError); updateErr != nil {
			log.Error("failed to update feed status to error", "feed_id", evt.FeedID, "error", updateErr.Error())
		}
		return err
	}

	if needsMetadataUpdate {
		if err := f.updateFeedMetadata(ctx, feed); err != nil {
			log.Error("failed to update feed metadata", "feed_id", evt.FeedID, "error", err.Error())
		}
	}

	log.Info("successfully completed feed fetch task", "feed_id", evt.FeedID, "articles_processed", len(articles))
	return nil
}

func (f *FeedFetcher) updateFeedMetadata(ctx context.Context, feed *models.Feed) error {
	log := logger.FromContext(ctx)
	log.Info("updating feed metadata", "feed_id", feed.ID, "url", feed.URL)

	parsedFeed, err := f.parser.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		log.Error("failed to parse feed for metadata update", "feed_id", feed.ID, "error", err.Error())
		return nil // articles already saved, skip metadata
	}

	title := parsedFeed.Title
	if title == "" {
		title = feed.URL
	}

	err = f.feedRepo.UpdateFeedMetadata(ctx, feed.ID, title, parsedFeed.Description, models.FeedStatusActive)
	if err != nil {
		log.Error("failed to save feed metadata", "feed_id", feed.ID, "error", err.Error())
		return err
	}

	log.Info("successfully updated feed metadata", "feed_id", feed.ID, "title", title)
	return nil
}
