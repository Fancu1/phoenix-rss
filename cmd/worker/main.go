package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	"github.com/Fancu1/phoenix-rss/internal/worker"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := logger.New(slog.LevelDebug)

	db := repository.InitDB(&cfg.Database)
	if err := repository.InitSchema(db); err != nil {
		logger.Error("failed to init schema", "error", err)
		os.Exit(1)
	}

	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)

	redisConnOpt := asynq.RedisClientOpt{
		Addr: cfg.Redis.Address,
	}
	articleService := core.NewArticleService(feedRepo, articleRepo, logger)
	taskProcessor := worker.NewTaskProcesser(logger, redisConnOpt, articleService)

	if err := taskProcessor.Start(); err != nil {
		logger.Error("Failed to start task processor", "error", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker server ...")
	taskProcessor.Stop()
}
