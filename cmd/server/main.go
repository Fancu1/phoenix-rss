package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	"github.com/Fancu1/phoenix-rss/internal/server"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
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

	redisConnOpt := asynq.RedisClientOpt{
		Addr: cfg.Redis.Address,
	}
	taskClient := asynq.NewClient(redisConnOpt)
	defer taskClient.Close()

	// Initialize repositories
	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize services
	feedSrv := core.NewFeedService(feedRepo, logger)
	articleSvc := core.NewArticleService(feedRepo, articleRepo, logger)
	userSvc := core.NewUserService(userRepo, cfg.Auth.JWTSecret)

	srv := server.New(cfg, logger, taskClient, feedSrv, articleSvc, userSvc, feedRepo)
	if err := srv.Start(); err != nil {
		logger.Error("failed to start server, error: %w", err)
		os.Exit(1)
	}
}
