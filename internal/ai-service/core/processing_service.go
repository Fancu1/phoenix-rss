package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Fancu1/phoenix-rss/internal/ai-service/client"
	article_eventspb "github.com/Fancu1/phoenix-rss/proto/gen/article_events"
)

// ProcessingService handle article processing using AI
type ProcessingService struct {
	llmClient client.LLMClientInterface
	logger    *slog.Logger
}

// NewProcessingService create a new processing service instance
func NewProcessingService(llmClient client.LLMClientInterface, logger *slog.Logger) *ProcessingService {
	return &ProcessingService{
		llmClient: llmClient,
		logger:    logger,
	}
}

// ProcessArticle process an article and returns the processed event
func (s *ProcessingService) ProcessArticle(ctx context.Context, event *article_eventspb.ArticlePersistedEvent) (*article_eventspb.ArticleProcessedEvent, error) {
	s.logger.Info("processing article",
		"article_id", event.ArticleId,
		"feed_id", event.FeedId,
		"title", event.Title,
	)

	startTime := time.Now()

	if event.ArticleId == 0 {
		return nil, fmt.Errorf("invalid article ID: %d", event.ArticleId)
	}

	if event.Title == "" && event.Content == "" {
		return nil, fmt.Errorf("both title and content are empty for article %d", event.ArticleId)
	}

	// Process article content with LLM
	result, err := s.llmClient.ProcessArticle(ctx, event.Title, event.Content)
	if err != nil {
		s.logger.Error("failed to process article with LLM",
			"article_id", event.ArticleId,
			"error", err,
		)
		return nil, fmt.Errorf("LLM processing failed: %w", err)
	}

	duration := time.Since(startTime)

	// Create processed event
	processedEvent := &article_eventspb.ArticleProcessedEvent{
		ArticleId:       event.ArticleId,
		Summary:         result.Summary,
		ProcessingModel: s.llmClient.GetModel(),
	}

	s.logger.Info("article processing completed",
		"article_id", event.ArticleId,
		"summary_length", len(result.Summary),
		"processing_duration", duration,
	)

	return processedEvent, nil
}

// ProcessBatch processes multiple articles in batch
func (s *ProcessingService) ProcessBatch(ctx context.Context, articles []*article_eventspb.ArticlePersistedEvent) ([]*article_eventspb.ArticleProcessedEvent, error) {
	if len(articles) == 0 {
		return []*article_eventspb.ArticleProcessedEvent{}, nil
	}

	s.logger.Info("processing article batch", "batch_size", len(articles))

	var results []*article_eventspb.ArticleProcessedEvent
	var errors []error

	for i, event := range articles {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		s.logger.Debug("processing article in batch",
			"batch_index", i+1,
			"batch_size", len(articles),
			"article_id", event.ArticleId,
		)

		result, err := s.ProcessArticle(ctx, event)
		if err != nil {
			s.logger.Error("failed to process article in batch",
				"article_id", event.ArticleId,
				"batch_index", i+1,
				"error", err,
			)
			errors = append(errors, err)
			continue
		}

		results = append(results, result)
	}

	if len(errors) > 0 {
		s.logger.Warn("batch processing completed with errors",
			"successful", len(results),
			"failed", len(errors),
			"total", len(articles),
		)
	} else {
		s.logger.Info("batch processing completed successfully",
			"processed", len(results),
			"total", len(articles),
		)
	}

	return results, nil
}
