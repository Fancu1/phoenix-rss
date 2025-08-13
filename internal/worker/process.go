package worker

import (
	"context"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/logger"
)

// EventHandler consume events and triggers article fetching
type EventHandler struct {
	logger         *slog.Logger
	articleService *core.ArticleService
}

func NewEventHandler(logger *slog.Logger, articleService *core.ArticleService) *EventHandler {
	return &EventHandler{
		logger:         logger,
		articleService: articleService,
	}
}

func (h *EventHandler) HandleFeedFetch(ctx context.Context, evt events.FeedFetchEvent) error {
	taskCtx := logger.WithValue(ctx, "feed_id", evt.FeedID)
	log := logger.FromContext(taskCtx)
	log.Info("starting feed fetch", "feed_id", evt.FeedID)
	articles, err := h.articleService.FetchAndSaveArticles(taskCtx, evt.FeedID)
	if err != nil {
		log.Error("failed to fetch and save articles for feed", "feed_id", evt.FeedID, "error", err.Error())
		return err
	}
	log.Info("successfully completed feed fetch task", "feed_id", evt.FeedID, "articles_processed", len(articles))
	return nil
}

type Worker struct {
	logger    *slog.Logger
	consumers []events.Consumer
}

func NewWorker(logger *slog.Logger) *Worker {
	return &Worker{
		logger: logger,
	}
}

func (w *Worker) RegisterConsumer(consumer events.Consumer) {
	w.consumers = append(w.consumers, consumer)
}

func (w *Worker) Start() error {
	g, ctx := errgroup.WithContext(context.Background())
	for _, consumer := range w.consumers {
		g.Go(func() error {
			w.logger.Info("starting consumer", "consumer", consumer)
			err := consumer.Start(ctx)
			if err != nil {
				w.logger.Error("failed to start consumer", "consumer", consumer, "error", err)
			}
			return err
		})
	}
	return g.Wait()
}

func (w *Worker) Stop() {
	for _, consumer := range w.consumers {
		w.logger.Info("stopping consumer", "consumer", consumer)
		consumer.Stop(context.Background())
	}
}
