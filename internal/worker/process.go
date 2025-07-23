package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/tasks"
)

type TaskProcesser struct {
	logger         *slog.Logger
	server         *asynq.Server
	articleService *core.ArticleService
}

func NewTaskProcesser(logger *slog.Logger, redisOpt asynq.RedisClientOpt, articleService *core.ArticleService) *TaskProcesser {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			Logger: NewSlogLogger(logger),
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.Error("Error processing task", "task_id", task.Type(), "payload", string(task.Payload()), "error", err)
			}),
		},
	)

	return &TaskProcesser{
		logger:         logger,
		server:         server,
		articleService: articleService,
	}
}

func (p *TaskProcesser) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TaskFeedFetch, p.HandleFeedFetchTask)

	return p.server.Start(mux)
}

func (p *TaskProcesser) Stop() {
	p.server.Stop()
	p.server.Shutdown()
}

func (p *TaskProcesser) HandleFeedFetchTask(ctx context.Context, task *asynq.Task) error {
	var payload tasks.FetchFeedPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	p.logger.Info("Start fetching feed", "feed_id", payload.FeedID)

	articles, err := p.articleService.FetchAndSaveArticles(ctx, payload.FeedID)
	if err != nil {
		p.logger.Error("Failed to fetch and save articles for feed",
			"feed_id", payload.FeedID,
			"error", err,
		)
		return err
	}

	p.logger.Info("Successfully fetched and saved articles for feed",
		"feed_id", payload.FeedID,
		"count", len(articles),
	)
	return nil
}

type SlogAdapter struct {
	logger *slog.Logger
}

func NewSlogLogger(logger *slog.Logger) *SlogAdapter {
	return &SlogAdapter{logger: logger}
}

func (l *SlogAdapter) Info(args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...))
}

func (l *SlogAdapter) Debug(args ...interface{}) {
	l.logger.Debug(fmt.Sprint(args...))
}

func (l *SlogAdapter) Warn(args ...interface{}) {
	l.logger.Warn(fmt.Sprint(args...))
}

func (l *SlogAdapter) Error(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
}

func (l *SlogAdapter) Fatal(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
}
