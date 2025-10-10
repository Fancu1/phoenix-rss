package worker

import (
	"context"
	"log/slog"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/core"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

type ArticleUpdateWorker struct {
	logger  *slog.Logger
	checker *core.ArticleUpdateChecker
}

func NewArticleUpdateWorker(logger *slog.Logger, checker *core.ArticleUpdateChecker) *ArticleUpdateWorker {
	return &ArticleUpdateWorker{logger: logger, checker: checker}
}

func (w *ArticleUpdateWorker) HandleArticleCheck(ctx context.Context, event events.ArticleCheckEvent) error {
	taskCtx := logger.WithValue(ctx, "article_id", event.ArticleID)
	taskCtx = logger.WithValue(taskCtx, "request_id", event.RequestID)
	return w.checker.HandleEvent(taskCtx, event)
}
