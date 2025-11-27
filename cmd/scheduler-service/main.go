package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/scheduler-service/client"
	"github.com/Fancu1/phoenix-rss/internal/scheduler-service/service"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

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

	log := logger.New(slog.LevelDebug)

	// Create gRPC connection to feed service
	conn, err := grpc.NewClient(
		cfg.FeedService.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("failed to connect to feed service", "address", cfg.FeedService.Address, "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Create feed service client
	feedClient := client.NewFeedServiceClient(conn, log)

	// Create Kafka producer
	producer := events.NewKafkaProducer(log, events.KafkaConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.FeedFetch.Topic,
		GroupID: cfg.Kafka.FeedFetch.FeedServiceGroupID, // Use same topic and group for scheduler
	})
	defer producer.Close()

	articleCheckProducer := events.NewKafkaArticleCheckProducer(log, events.KafkaConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.ArticleCheck.Topic,
	})
	defer articleCheckProducer.Close()

	// Parse batch delay duration
	batchDelay, err := time.ParseDuration(cfg.SchedulerService.BatchDelay)
	if err != nil {
		log.Error("failed to parse batch delay", "batch_delay", cfg.SchedulerService.BatchDelay, "error", err)
		os.Exit(1)
	}

	minCheckInterval, err := time.ParseDuration(cfg.SchedulerService.ArticleCheck.MinCheckInterval)
	if err != nil {
		log.Error("failed to parse article check min interval", "value", cfg.SchedulerService.ArticleCheck.MinCheckInterval, "error", err)
		os.Exit(1)
	}

	articleWindow := time.Duration(cfg.SchedulerService.ArticleCheck.WindowDays) * 24 * time.Hour
	articlePageSize := cfg.SchedulerService.ArticleCheck.PageSize
	if articlePageSize <= 0 {
		log.Error("invalid article check page size", "value", articlePageSize)
		os.Exit(1)
	}

	// Create and start scheduler
	scheduler := service.NewScheduler(
		log,
		feedClient,
		producer,
		articleCheckProducer,
		cfg.SchedulerService.Schedule,
		cfg.SchedulerService.BatchSize,
		batchDelay,
		cfg.SchedulerService.MaxConcurrent,
		cfg.SchedulerService.ArticleCheck.Cron,
		articleWindow,
		minCheckInterval,
		articlePageSize,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info("starting scheduler service",
		"schedule", cfg.SchedulerService.Schedule,
		"batch_size", cfg.SchedulerService.BatchSize,
		"batch_delay", cfg.SchedulerService.BatchDelay,
		"max_concurrent", cfg.SchedulerService.MaxConcurrent,
	)

	// Start scheduler
	if err := scheduler.Start(ctx); err != nil {
		log.Error("failed to start scheduler", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	select {
	case sig := <-signalChan:
		log.Info("received shutdown signal", "signal", sig)
		cancel()
	case <-ctx.Done():
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := scheduler.Stop(shutdownCtx); err != nil {
		log.Error("failed to stop scheduler gracefully", "error", err)
	}

	log.Info("scheduler service shutdown completed")
}
