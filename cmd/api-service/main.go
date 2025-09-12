package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/internal/api-service/server"
	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

//go:embed all:dist
var staticFiles embed.FS

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	appLogger := logger.New(slog.LevelDebug)

	feedSvc, err := core.NewFeedServiceClient(cfg.FeedService.Address)
	if err != nil {
		appLogger.Error("failed to connect to feed service", "address", cfg.FeedService.Address, "error", err)
		os.Exit(1)
	}
	defer feedSvc.Close()

	articleSvc, err := core.NewArticleServiceClient(cfg.FeedService.Address)
	if err != nil {
		appLogger.Error("failed to connect to feed service for articles", "address", cfg.FeedService.Address, "error", err)
		os.Exit(1)
	}
	defer articleSvc.Close()

	userSvc, err := core.NewUserServiceClient(cfg.UserService.Address)
	if err != nil {
		appLogger.Error("failed to connect to user service", "address", cfg.UserService.Address, "error", err)
		os.Exit(1)
	}
	defer userSvc.Close()

	srv, err := server.New(cfg, appLogger, feedSvc, articleSvc, userSvc, staticFiles)
	if err != nil {
		appLogger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := srv.Start(); err != nil {
		appLogger.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
