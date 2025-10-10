package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Fancu1/phoenix-rss/internal/events"
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

func (m *MockFeedClient) ListArticlesToCheck(ctx context.Context, timeRange models.ArticleCheckWindow, pageSize int, pageToken string) (*models.ArticleCheckPage, error) {
	args := m.Called(ctx, timeRange, pageSize, pageToken)
	var page *models.ArticleCheckPage
	if v := args.Get(0); v != nil {
		page = v.(*models.ArticleCheckPage)
	}
	return page, args.Error(1)
}

// MockProducer implements a mock Kafka producer
type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) PublishFeedFetch(ctx context.Context, feedID uint) error {
	args := m.Called(ctx, feedID)
	return args.Error(0)
}

type MockArticleCheckProducer struct {
	mock.Mock
}

func (m *MockArticleCheckProducer) PublishArticleCheck(ctx context.Context, event events.ArticleCheckEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestScheduler_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 10, 1*time.Second, 2, "", 24*time.Hour, 4*time.Hour, 100)

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

	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 10, 1*time.Second, 2, "", 24*time.Hour, 4*time.Hour, 100)

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

	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 10, 1*time.Second, 2, "", 24*time.Hour, 4*time.Hour, 100)

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

	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 10, 1*time.Second, 2, "", 24*time.Hour, 4*time.Hour, 100)

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

	scheduler := NewScheduler(logger, mockClient, mockProducer, nil, "@every 1h", 10, 1*time.Second, 2, "", 24*time.Hour, 4*time.Hour, 100)

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

func TestScheduler_TriggerArticleChecks_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)
	mockArticleProducer := new(MockArticleCheckProducer)

	articles := &models.ArticleCheckPage{
		Items: []*models.ArticleToCheck{
			{ArticleID: 1, FeedID: 2, URL: "https://example.com/a1", PrevETag: "etag"},
			{ArticleID: 2, FeedID: 3, URL: "https://example.com/a2"},
		},
	}

	scheduler := NewScheduler(logger, mockClient, mockProducer, mockArticleProducer, "@every 1h", 10, 1*time.Second, 2, "0 */2 * * * *", 7*24*time.Hour, 4*time.Hour, 50)

	ctx := context.Background()
	mockClient.
		On("ListArticlesToCheck", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("models.ArticleCheckWindow"), 50, "").
		Return(articles, nil)

	mockArticleProducer.
		On("PublishArticleCheck", mock.AnythingOfType("*context.valueCtx"), mock.MatchedBy(func(evt events.ArticleCheckEvent) bool {
			return evt.ArticleID == 1 && evt.URL == "https://example.com/a1"
		})).
		Return(nil)
	mockArticleProducer.
		On("PublishArticleCheck", mock.AnythingOfType("*context.valueCtx"), mock.MatchedBy(func(evt events.ArticleCheckEvent) bool {
			return evt.ArticleID == 2 && evt.URL == "https://example.com/a2"
		})).
		Return(nil)

	scheduler.triggerArticleChecks(ctx)

	mockClient.AssertExpectations(t)
	mockArticleProducer.AssertExpectations(t)
}

func TestScheduler_TriggerArticleChecks_Error(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := new(MockFeedClient)
	mockProducer := new(MockProducer)
	mockArticleProducer := new(MockArticleCheckProducer)

	scheduler := NewScheduler(logger, mockClient, mockProducer, mockArticleProducer, "@every 1h", 10, 1*time.Second, 2, "0 */2 * * * *", 7*24*time.Hour, 4*time.Hour, 50)

	ctx := context.Background()
	mockClient.
		On("ListArticlesToCheck", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("models.ArticleCheckWindow"), 50, "").
		Return((*models.ArticleCheckPage)(nil), assert.AnError)

	scheduler.triggerArticleChecks(ctx)

	mockClient.AssertExpectations(t)
	mockArticleProducer.AssertNotCalled(t, "PublishArticleCheck", mock.Anything, mock.Anything)
}
