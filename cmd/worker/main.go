package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/events"
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

	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)

	articleService := core.NewArticleService(feedRepo, articleRepo, logger)
	eventHandler := worker.NewEventHandler(logger, articleService)

	appWorker := worker.NewWorker(logger)
	feedFetchConsumer := events.NewKafkaConsumer(logger, events.KafkaConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		GroupID: cfg.Kafka.GroupID,
	}, eventHandler.HandleFeedFetch)
	appWorker.RegisterConsumer(feedFetchConsumer)

	go func() {
		if err := appWorker.Start(); err != nil {
			logger.Error("Failed to start worker", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker server ...")
	appWorker.Stop()
}
