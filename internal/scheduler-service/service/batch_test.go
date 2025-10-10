package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Fancu1/phoenix-rss/internal/scheduler-service/models"
)

func TestScheduler_CreateBatches(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 3, 1*time.Second, 2, "", 24*time.Hour, 4*time.Hour, 100)

	// Test with 7 feeds and batch size of 3
	feeds := []*models.Feed{
		{ID: 1, Title: "Feed 1"},
		{ID: 2, Title: "Feed 2"},
		{ID: 3, Title: "Feed 3"},
		{ID: 4, Title: "Feed 4"},
		{ID: 5, Title: "Feed 5"},
		{ID: 6, Title: "Feed 6"},
		{ID: 7, Title: "Feed 7"},
	}

	batches := scheduler.createBatches(feeds)

	// Should create 3 batches: [1,2,3], [4,5,6], [7]
	assert.Len(t, batches, 3)
	assert.Len(t, batches[0], 3)
	assert.Len(t, batches[1], 3)
	assert.Len(t, batches[2], 1)

	// Verify content
	assert.Equal(t, uint(1), batches[0][0].ID)
	assert.Equal(t, uint(3), batches[0][2].ID)
	assert.Equal(t, uint(4), batches[1][0].ID)
	assert.Equal(t, uint(6), batches[1][2].ID)
	assert.Equal(t, uint(7), batches[2][0].ID)
}

func TestScheduler_CreateBatches_EmptyFeeds(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 10, 1*time.Second, 2, "", 24*time.Hour, 4*time.Hour, 100)

	feeds := []*models.Feed{}
	batches := scheduler.createBatches(feeds)

	assert.Len(t, batches, 0)
}

func TestScheduler_ProcessBatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 10, 1*time.Second, 2, "", 24*time.Hour, 4*time.Hour, 100)

	// Setup mock expectations
	feeds := []*models.Feed{
		{ID: 1, Title: "Feed 1", URL: "http://example.com/feed1"},
		{ID: 2, Title: "Feed 2", URL: "http://example.com/feed2"},
	}

	ctx := context.Background()
	mockProducer.On("PublishFeedFetch", mock.AnythingOfType("*context.valueCtx"), uint(1)).Return(nil)
	mockProducer.On("PublishFeedFetch", mock.AnythingOfType("*context.valueCtx"), uint(2)).Return(nil)

	// Test processing a batch
	successCount, failedCount := scheduler.processBatch(ctx, feeds)

	// Verify results
	assert.Equal(t, 2, successCount)
	assert.Equal(t, 0, failedCount)

	// Verify all expectations were met
	mockProducer.AssertExpectations(t)
}

func TestScheduler_ProcessBatch_WithFailures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 10, 1*time.Second, 2, "", 24*time.Hour, 4*time.Hour, 100)

	// Setup mock expectations with one failure
	feeds := []*models.Feed{
		{ID: 1, Title: "Feed 1", URL: "http://example.com/feed1"},
		{ID: 2, Title: "Feed 2", URL: "http://example.com/feed2"},
	}

	ctx := context.Background()
	mockProducer.On("PublishFeedFetch", mock.AnythingOfType("*context.valueCtx"), uint(1)).Return(nil)
	mockProducer.On("PublishFeedFetch", mock.AnythingOfType("*context.valueCtx"), uint(2)).Return(assert.AnError)

	// Test processing a batch with failures
	successCount, failedCount := scheduler.processBatch(ctx, feeds)

	// Verify results
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 1, failedCount)

	// Verify all expectations were met
	mockProducer.AssertExpectations(t)
}

func TestScheduler_BatchProcessing_Integration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	// Use small batch size and delay for testing
	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 2, 10*time.Millisecond, 1, "", 24*time.Hour, 4*time.Hour, 100)

	// Setup mock expectations
	feeds := []*models.Feed{
		{ID: 1, Title: "Feed 1", URL: "http://example.com/feed1"},
		{ID: 2, Title: "Feed 2", URL: "http://example.com/feed2"},
		{ID: 3, Title: "Feed 3", URL: "http://example.com/feed3"},
		{ID: 4, Title: "Feed 4", URL: "http://example.com/feed4"},
	}

	ctx := context.Background()
	mockClient.On("GetAllFeeds", mock.AnythingOfType("*context.valueCtx")).Return(feeds, nil)

	// Expect all feeds to be processed
	for _, feed := range feeds {
		mockProducer.On("PublishFeedFetch", mock.AnythingOfType("*context.valueCtx"), feed.ID).Return(nil)
	}

	// Record start time
	startTime := time.Now()

	// Trigger feed fetches
	scheduler.triggerFeedFetches(ctx)

	// Should take at least one batch delay (10ms) since we have 2 batches
	elapsed := time.Since(startTime)
	assert.GreaterOrEqual(t, elapsed, 10*time.Millisecond)

	// Verify all expectations were met
	mockClient.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}
