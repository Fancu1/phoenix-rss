package core

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Fancu1/phoenix-rss/internal/ai-service/client"
	article_eventspb "github.com/Fancu1/phoenix-rss/proto/gen/article_events"
)

// MockLLMClient is a mock implementation of LLMClientInterface for testing
type MockLLMClient struct {
	shouldError bool
	result      *client.ProcessingResult
	model       string
}

func (m *MockLLMClient) ProcessArticle(ctx context.Context, title, content string) (*client.ProcessingResult, error) {
	if m.shouldError {
		return nil, errors.New("mock LLM error")
	}
	return m.result, nil
}

func (m *MockLLMClient) GetModel() string {
	return m.model
}

func TestProcessingService_ProcessArticle(t *testing.T) {
	tests := []struct {
		name        string
		event       *article_eventspb.ArticlePersistedEvent
		mockResult  *client.ProcessingResult
		mockError   bool
		expectError bool
	}{
		{
			name: "successful processing",
			event: &article_eventspb.ArticlePersistedEvent{
				ArticleId:   1,
				FeedId:      1,
				Title:       "Test Article",
				Content:     "This is test content",
				Url:         "http://example.com/article",
				Description: "Test description",
				PublishedAt: time.Now().Unix(),
			},
			mockResult: &client.ProcessingResult{
				Summary: "Test summary",
			},
			mockError:   false,
			expectError: false,
		},
		{
			name: "invalid article ID",
			event: &article_eventspb.ArticlePersistedEvent{
				ArticleId: 0, // Invalid ID
				Title:     "Test Article",
				Content:   "Test content",
			},
			expectError: true,
		},
		{
			name: "empty title and content",
			event: &article_eventspb.ArticlePersistedEvent{
				ArticleId: 1,
				Title:     "",
				Content:   "",
			},
			expectError: true,
		},
		{
			name: "LLM processing error",
			event: &article_eventspb.ArticlePersistedEvent{
				ArticleId: 1,
				Title:     "Test Article",
				Content:   "Test content",
			},
			mockError:   true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLM client
			mockClient := &MockLLMClient{
				shouldError: tt.mockError,
				result:      tt.mockResult,
				model:       "test-model",
			}

			// Create processing service
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			service := NewProcessingService(mockClient, logger)

			// Test
			ctx := context.Background()
			result, err := service.ProcessArticle(ctx, tt.event)

			// Verify
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result, but got nil")
				return
			}

			// Verify result fields
			if result.ArticleId != tt.event.ArticleId {
				t.Errorf("Expected ArticleId: %d, got: %d", tt.event.ArticleId, result.ArticleId)
			}

			if tt.mockResult != nil {
				if result.Summary != tt.mockResult.Summary {
					t.Errorf("Expected Summary: %s, got: %s", tt.mockResult.Summary, result.Summary)
				}

				if result.ProcessingModel != "test-model" {
					t.Errorf("Expected ProcessingModel: test-model, got: %s", result.ProcessingModel)
				}
			}
		})
	}
}

func TestProcessingService_ProcessBatch(t *testing.T) {
	// Create mock LLM client
	mockClient := &MockLLMClient{
		shouldError: false,
		result: &client.ProcessingResult{
			Summary: "Test summary",
		},
		model: "test-model",
	}

	// Create processing service
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	service := NewProcessingService(mockClient, logger)

	t.Run("empty batch", func(t *testing.T) {
		ctx := context.Background()
		results, err := service.ProcessBatch(ctx, []*article_eventspb.ArticlePersistedEvent{})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})

	t.Run("successful batch processing", func(t *testing.T) {
		articles := []*article_eventspb.ArticlePersistedEvent{
			{
				ArticleId: 1,
				Title:     "Article 1",
				Content:   "Content 1",
			},
			{
				ArticleId: 2,
				Title:     "Article 2",
				Content:   "Content 2",
			},
		}

		ctx := context.Background()
		results, err := service.ProcessBatch(ctx, articles)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		// Verify individual results
		for i, result := range results {
			if result.ArticleId != articles[i].ArticleId {
				t.Errorf("Result %d: Expected ArticleId %d, got %d", i, articles[i].ArticleId, result.ArticleId)
			}
		}
	})

	t.Run("batch with some failures", func(t *testing.T) {
		articles := []*article_eventspb.ArticlePersistedEvent{
			{
				ArticleId: 1,
				Title:     "Article 1",
				Content:   "Content 1",
			},
			{
				ArticleId: 0, // Invalid ID should cause error
				Title:     "Article 2",
				Content:   "Content 2",
			},
			{
				ArticleId: 3,
				Title:     "Article 3",
				Content:   "Content 3",
			},
		}

		ctx := context.Background()
		results, err := service.ProcessBatch(ctx, articles)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should have 2 successful results (skipping the invalid one)
		if len(results) != 2 {
			t.Errorf("Expected 2 successful results, got %d", len(results))
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		articles := []*article_eventspb.ArticlePersistedEvent{
			{ArticleId: 1, Title: "Article 1", Content: "Content 1"},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		results, err := service.ProcessBatch(ctx, articles)

		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}

		// Should have no results due to cancellation
		if len(results) != 0 {
			t.Errorf("Expected 0 results due to cancellation, got %d", len(results))
		}
	})
}
