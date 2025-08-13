package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/logger"
	"github.com/Fancu1/phoenix-rss/internal/repository"
	"github.com/Fancu1/phoenix-rss/internal/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := logger.New(slog.LevelDebug)

	db := repository.InitDB(&cfg.Database)

	// Initialize Kafka producer
	producer := events.NewKafkaProducer(logger, events.KafkaConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		GroupID: cfg.Kafka.GroupID,
	})
	defer producer.Close()

	// Initialize repositories
	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize services
	feedSrv := core.NewFeedService(feedRepo, logger)
	articleSvc := core.NewArticleService(feedRepo, articleRepo, logger)
	userSvc := core.NewUserService(userRepo, cfg.Auth.JWTSecret)

	srv := server.New(cfg, logger, producer, feedSrv, articleSvc, userSvc, feedRepo)
	if err := srv.Start(); err != nil {
		logger.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
