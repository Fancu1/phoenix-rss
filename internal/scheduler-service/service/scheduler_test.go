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

// MockFeedClient implements a mock feed service client
type MockFeedClient struct {
	mock.Mock
}

func (m *MockFeedClient) GetAllFeeds(ctx context.Context) ([]*models.Feed, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.Feed), args.Error(1)
}

// MockProducer implements a mock Kafka producer
type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) PublishFeedFetch(ctx context.Context, feedID uint) error {
	args := m.Called(ctx, feedID)
	return args.Error(0)
}

func TestScheduler_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, "@every 1h", 10, 1*time.Second, 2)

	// Test initial state
	assert.False(t, scheduler.IsRunning())

	// Test start
	ctx := context.Background()
	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Test double start should fail
	err = scheduler.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = scheduler.Stop(stopCtx)
	assert.NoError(t, err)
	assert.False(t, scheduler.IsRunning())

	// Test double stop should be safe
	err = scheduler.Stop(stopCtx)
	assert.NoError(t, err)
}

func TestScheduler_TriggerFeedFetches_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, "@every 1h", 10, 1*time.Second, 2)

	// Setup mock expectations
	feeds := []*models.Feed{
		{ID: 1, Title: "Feed 1", URL: "http://example.com/feed1"},
		{ID: 2, Title: "Feed 2", URL: "http://example.com/feed2"},
	}

	ctx := context.Background()
	mockClient.On("GetAllFeeds", mock.AnythingOfType("*context.valueCtx")).Return(feeds, nil)
	mockProducer.On("PublishFeedFetch", mock.AnythingOfType("*context.valueCtx"), uint(1)).Return(nil)
	mockProducer.On("PublishFeedFetch", mock.AnythingOfType("*context.valueCtx"), uint(2)).Return(nil)

	// Test the trigger function
	scheduler.triggerFeedFetches(ctx)

	// Verify all expectations were met
	mockClient.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

func TestScheduler_TriggerFeedFetches_NoFeeds(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, "@every 1h", 10, 1*time.Second, 2)

	// Setup mock expectations
	feeds := []*models.Feed{}

	ctx := context.Background()
	mockClient.On("GetAllFeeds", mock.AnythingOfType("*context.valueCtx")).Return(feeds, nil)

	// Test the trigger function
	scheduler.triggerFeedFetches(ctx)

	// Verify expectations
	mockClient.AssertExpectations(t)
	// Producer should not be called when there are no feeds
	mockProducer.AssertNotCalled(t, "PublishFeedFetch")
}

func TestScheduler_TriggerFeedFetches_GetFeedsError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, "@every 1h", 10, 1*time.Second, 2)

	// Setup mock expectations
	ctx := context.Background()
	mockClient.On("GetAllFeeds", mock.AnythingOfType("*context.valueCtx")).Return(([]*models.Feed)(nil), assert.AnError)

	// Test the trigger function
	scheduler.triggerFeedFetches(ctx)

	// Verify expectations
	mockClient.AssertExpectations(t)
	// Producer should not be called when GetAllFeeds fails
	mockProducer.AssertNotCalled(t, "PublishFeedFetch")
}

func TestScheduler_TriggerFeedFetches_PartialFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, "@every 1h", 10, 1*time.Second, 2)

	// Setup mock expectations
	feeds := []*models.Feed{
		{ID: 1, Title: "Feed 1", URL: "http://example.com/feed1"},
		{ID: 2, Title: "Feed 2", URL: "http://example.com/feed2"},
	}

	ctx := context.Background()
	mockClient.On("GetAllFeeds", mock.AnythingOfType("*context.valueCtx")).Return(feeds, nil)
	mockProducer.On("PublishFeedFetch", mock.AnythingOfType("*context.valueCtx"), uint(1)).Return(nil)
	mockProducer.On("PublishFeedFetch", mock.AnythingOfType("*context.valueCtx"), uint(2)).Return(assert.AnError)

	// Test the trigger function
	scheduler.triggerFeedFetches(ctx)

	// Verify expectations
	mockClient.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}
