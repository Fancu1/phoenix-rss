package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Fancu1/phoenix-rss/internal/ai-service/client"
	"github.com/Fancu1/phoenix-rss/internal/ai-service/core"
	"github.com/Fancu1/phoenix-rss/internal/ai-service/worker"
	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(slog.LevelDebug)

	requestTimeout, err := time.ParseDuration(cfg.AIService.RequestTimeout)
	if err != nil {
		log.Error("failed to parse request timeout", "timeout", cfg.AIService.RequestTimeout, "error", err)
		os.Exit(1)
	}

	// Create LLM client
	llmClient := client.NewLLMClient(
		cfg.AIService.LLMBaseURL,
		cfg.AIService.LLMAPIKey,
		cfg.AIService.LLMModel,
		requestTimeout,
		log,
	)

	// Create processing service
	processingService := core.NewProcessingService(llmClient, log)

	// Create and start article processor
	articleProcessor := worker.NewArticleProcessor(
		log,
		processingService,
		cfg.Kafka.Brokers,
		cfg.Kafka.AIProcessing.AIServiceGroupID,
		cfg.Kafka.AIProcessing.ArticlesNewTopic,
		cfg.Kafka.AIProcessing.ArticlesProcessedTopic,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info("starting AI service",
		"llm_model", cfg.AIService.LLMModel,
		"request_timeout", cfg.AIService.RequestTimeout,
		"articles_new_topic", cfg.Kafka.AIProcessing.ArticlesNewTopic,
		"articles_processed_topic", cfg.Kafka.AIProcessing.ArticlesProcessedTopic,
	)

	// Start article processor
	go func() {
		if err := articleProcessor.Start(ctx); err != nil && err != context.Canceled {
			log.Error("article processor failed", "error", err)
			cancel()
		}
	}()

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

	if err := articleProcessor.Stop(shutdownCtx); err != nil {
		log.Error("failed to stop article processor gracefully", "error", err)
	}

	log.Info("AI service shutdown completed")
}
