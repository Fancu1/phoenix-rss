package worker

import (
	"context"
	"log/slog"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/core"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

// FeedFetcher consumes events and triggers article fetching
type FeedFetcher struct {
	logger         *slog.Logger
	articleService *core.ArticleService
}

// NewFeedFetcher creates a new feed fetcher instance
func NewFeedFetcher(logger *slog.Logger, articleService *core.ArticleService) *FeedFetcher {
	return &FeedFetcher{
		logger:         logger,
		articleService: articleService,
	}
}

// HandleFeedFetch handles feed fetch events
func (f *FeedFetcher) HandleFeedFetch(ctx context.Context, evt events.FeedFetchEvent) error {
	taskCtx := logger.WithValue(ctx, "feed_id", evt.FeedID)
	log := logger.FromContext(taskCtx)
	log.Info("starting feed fetch", "feed_id", evt.FeedID)
	articles, err := f.articleService.FetchAndSaveArticles(taskCtx, evt.FeedID)
	if err != nil {
		log.Error("failed to fetch and save articles for feed", "feed_id", evt.FeedID, "error", err.Error())
		return err
	}
	log.Info("successfully completed feed fetch task", "feed_id", evt.FeedID, "articles_processed", len(articles))
	return nil
}
