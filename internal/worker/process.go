package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/tasks"
)

type TaskProcesser struct {
	logger         *slog.Logger
	server         *asynq.Server
	articleService *core.ArticleService
}

func NewTaskProcesser(loggerInstance *slog.Logger, redisOpt asynq.RedisClientOpt, articleService *core.ArticleService) *TaskProcesser {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			Logger: NewSlogLogger(loggerInstance),
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				// Try to get task ID from context (available in asynq)
				taskID, hasTaskID := asynq.GetTaskID(ctx)
				if hasTaskID {
					// Create contextual logger with task information
					taskCtx := logger.WithTaskID(ctx, taskID)
					log := logger.FromContext(taskCtx)
					log.Error("Error processing task", "task_type", task.Type(), "payload", string(task.Payload()), "error", err)
				} else {
					// Fallback to basic logging if task ID not available
					loggerInstance.Error("Error processing task", "task_type", task.Type(), "payload", string(task.Payload()), "error", err)
				}
			}),
		},
	)

	return &TaskProcesser{
		logger:         loggerInstance,
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
	// Get task ID from asynq context (available in task handlers)
	taskID, hasTaskID := asynq.GetTaskID(ctx)
	var taskCtx context.Context
	var log *slog.Logger

	if hasTaskID {
		// Create enhanced context with task ID for tracing
		taskCtx = logger.WithTaskID(ctx, taskID)
		log = logger.FromContext(taskCtx)
	} else {
		// Fallback if task ID not available
		taskCtx = ctx
		log = p.logger
	}

	log.Info("processing feed fetch task", "task_type", task.Type())

	var payload tasks.FetchFeedPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Error("failed to unmarshal task payload", "error", err.Error(), "payload", string(task.Payload()))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Add feed ID to context for better tracing
	taskCtx = logger.WithValue(taskCtx, "feed_id", payload.FeedID)
	log = logger.FromContext(taskCtx)

	log.Info("starting feed fetch", "feed_id", payload.FeedID)

	articles, err := p.articleService.FetchAndSaveArticles(taskCtx, payload.FeedID)
	if err != nil {
		log.Error("failed to fetch and save articles for feed",
			"feed_id", payload.FeedID,
			"error", err.Error(),
		)
		return err
	}

	log.Info("successfully completed feed fetch task",
		"feed_id", payload.FeedID,
		"articles_processed", len(articles),
	)
	return nil
}

// WithValue adds a key-value pair to the context for logging
// This is a helper function for adding additional context to tasks
func WithValue(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
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
