package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/Fancu1/phoenix-rss/internal/api-service/core"
	"github.com/Fancu1/phoenix-rss/internal/api-service/server"
	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

//go:embed all:dist
var staticFiles embed.FS

func main() {
	if err := logger.InitFromEnv(); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

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

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.Address})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		appLogger.Warn("redis ping failed, token cache will be best-effort", "address", cfg.Redis.Address, "error", err)
	}
	cancel()
	defer redisClient.Close()

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		appLogger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	srv, err := server.New(cfg, db, feedSvc, articleSvc, userSvc, redisClient, staticFiles)
	if err != nil {
		appLogger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := srv.Start(); err != nil {
		appLogger.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
