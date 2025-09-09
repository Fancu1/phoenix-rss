package worker

import (
	"context"
	"log/slog"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/core"
	article_eventspb "github.com/Fancu1/phoenix-rss/proto/gen/article_events"
)

// AIResultHandler handles AI processing results for the feed service
type AIResultHandler struct {
	logger         *slog.Logger
	articleService core.ArticleServiceInterface
	eventConsumer  events.ArticleEventConsumer
}

// NewAIResultHandler creates a new AI result handler instance
func NewAIResultHandler(
	logger *slog.Logger,
	articleService core.ArticleServiceInterface,
	eventConsumer events.ArticleEventConsumer,
) *AIResultHandler {
	return &AIResultHandler{
		logger:         logger,
		articleService: articleService,
		eventConsumer:  eventConsumer,
	}
}

// Start begins processing AI results
func (h *AIResultHandler) Start(ctx context.Context) error {
	h.logger.Info("starting AI result handler for feed service")

	// start consuming ArticleProcessedEvent messages
	return h.eventConsumer.StartProcessedEventConsumer(ctx, h.handleArticleProcessed)
}

// Stop gracefully stops the AI result handler
func (h *AIResultHandler) Stop(ctx context.Context) error {
	h.logger.Info("stopping AI result handler for feed service")
	return h.eventConsumer.Stop(ctx)
}

// handleArticleProcessed handles an ArticleProcessedEvent
func (h *AIResultHandler) handleArticleProcessed(ctx context.Context, event *article_eventspb.ArticleProcessedEvent) error {
	h.logger.Debug("received AI processed article event",
		"article_id", event.ArticleId,
		"summary_length", len(event.Summary),
	)

	// Delegate to the article service
	if err := h.articleService.HandleArticleProcessed(ctx, event); err != nil {
		h.logger.Error("failed to handle AI processed article event",
			"article_id", event.ArticleId,
			"error", err,
		)
		return err
	}

	h.logger.Info("successfully handled AI processed article event",
		"article_id", event.ArticleId,
	)

	return nil
}
