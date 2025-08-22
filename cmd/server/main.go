package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/server"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := logger.New(slog.LevelDebug)

	feedSvc, err := core.NewFeedServiceClient(cfg.FeedService.Address)
	if err != nil {
		logger.Error("failed to connect to feed service", "address", cfg.FeedService.Address, "error", err)
		os.Exit(1)
	}
	defer feedSvc.Close()

	articleSvc, err := core.NewArticleServiceClient(cfg.FeedService.Address)
	if err != nil {
		logger.Error("failed to connect to feed service for articles", "address", cfg.FeedService.Address, "error", err)
		os.Exit(1)
	}
	defer articleSvc.Close()

	userSvc, err := core.NewUserServiceClient(cfg.UserService.Address)
	if err != nil {
		logger.Error("failed to connect to user service", "address", cfg.UserService.Address, "error", err)
		os.Exit(1)
	}
	defer userSvc.Close()

	srv := server.New(cfg, logger, feedSvc, articleSvc, userSvc)
	if err := srv.Start(); err != nil {
		logger.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
