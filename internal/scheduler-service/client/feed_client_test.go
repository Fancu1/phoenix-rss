package client

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Fancu1/phoenix-rss/internal/scheduler-service/models"
	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
)

// MockFeedServiceClient implements a mock gRPC client with correct signatures
type MockFeedServiceClient struct {
	feeds     []*feedpb.Feed
	articles  []*feedpb.ArticleToCheck
	nextToken string
	err       error
}

func (m *MockFeedServiceClient) ListAllFeeds(ctx context.Context, req *feedpb.ListAllFeedsRequest, opts ...grpc.CallOption) (*feedpb.ListAllFeedsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &feedpb.ListAllFeedsResponse{Feeds: m.feeds}, nil
}

func (m *MockFeedServiceClient) ListArticlesToCheck(ctx context.Context, req *feedpb.ListArticlesToCheckRequest, opts ...grpc.CallOption) (*feedpb.ListArticlesToCheckResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &feedpb.ListArticlesToCheckResponse{Items: m.articles, NextPageToken: m.nextToken}, nil
}

func (m *MockFeedServiceClient) SubscribeToFeed(ctx context.Context, req *feedpb.SubscribeToFeedRequest, opts ...grpc.CallOption) (*feedpb.SubscribeToFeedResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *MockFeedServiceClient) ListUserFeeds(ctx context.Context, req *feedpb.ListUserFeedsRequest, opts ...grpc.CallOption) (*feedpb.ListUserFeedsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *MockFeedServiceClient) UnsubscribeFromFeed(ctx context.Context, req *feedpb.UnsubscribeFromFeedRequest, opts ...grpc.CallOption) (*feedpb.UnsubscribeFromFeedResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *MockFeedServiceClient) ListArticles(ctx context.Context, req *feedpb.ListArticlesRequest, opts ...grpc.CallOption) (*feedpb.ListArticlesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *MockFeedServiceClient) GetArticle(ctx context.Context, req *feedpb.GetArticleRequest, opts ...grpc.CallOption) (*feedpb.GetArticleResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *MockFeedServiceClient) TriggerFetch(ctx context.Context, req *feedpb.TriggerFetchRequest, opts ...grpc.CallOption) (*feedpb.TriggerFetchResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *MockFeedServiceClient) CheckSubscription(ctx context.Context, req *feedpb.CheckSubscriptionRequest, opts ...grpc.CallOption) (*feedpb.CheckSubscriptionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *MockFeedServiceClient) BatchSubscribeToFeeds(ctx context.Context, req *feedpb.BatchSubscribeToFeedsRequest, opts ...grpc.CallOption) (*feedpb.BatchSubscribeToFeedsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestFeedServiceClient_GetAllFeeds_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Setup mock client with test data
	pbFeeds := []*feedpb.Feed{
		{
			Id:          1,
			Title:       "Test Feed 1",
			Url:         "http://example.com/feed1",
			Description: "Description 1",
			CreatedAt:   time.Now().Format(time.RFC3339),
			UpdatedAt:   time.Now().Format(time.RFC3339),
		},
		{
			Id:          2,
			Title:       "Test Feed 2",
			Url:         "http://example.com/feed2",
			Description: "Description 2",
			CreatedAt:   time.Now().Format(time.RFC3339),
			UpdatedAt:   time.Now().Format(time.RFC3339),
		},
	}

	mockClient := &MockFeedServiceClient{feeds: pbFeeds}

	// Create client with mock
	client := &FeedServiceClient{
		client: mockClient,
		logger: logger,
	}

	// Test GetAllFeeds
	ctx := context.Background()
	feeds, err := client.GetAllFeeds(ctx)

	// Assertions
	require.NoError(t, err)
	assert.Len(t, feeds, 2)

	// Verify conversion from protobuf to model
	assert.Equal(t, uint(1), feeds[0].ID)
	assert.Equal(t, "Test Feed 1", feeds[0].Title)
	assert.Equal(t, "http://example.com/feed1", feeds[0].URL)
	assert.Equal(t, "Description 1", feeds[0].Description)

	assert.Equal(t, uint(2), feeds[1].ID)
	assert.Equal(t, "Test Feed 2", feeds[1].Title)
	assert.Equal(t, "http://example.com/feed2", feeds[1].URL)
	assert.Equal(t, "Description 2", feeds[1].Description)
}

func TestFeedServiceClient_GetAllFeeds_Empty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Setup mock client with no feeds
	mockClient := &MockFeedServiceClient{feeds: []*feedpb.Feed{}}

	client := &FeedServiceClient{
		client: mockClient,
		logger: logger,
	}

	// Test GetAllFeeds
	ctx := context.Background()
	feeds, err := client.GetAllFeeds(ctx)

	// Assertions
	require.NoError(t, err)
	assert.Len(t, feeds, 0)
}

func TestFeedServiceClient_GetAllFeeds_Error(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Setup mock client with error
	mockClient := &MockFeedServiceClient{
		err: status.Error(codes.Internal, "internal server error"),
	}

	client := &FeedServiceClient{
		client: mockClient,
		logger: logger,
	}

	// Test GetAllFeeds
	ctx := context.Background()
	feeds, err := client.GetAllFeeds(ctx)

	// Assertions
	require.Error(t, err)
	assert.Nil(t, feeds)
	assert.Contains(t, err.Error(), "failed to list all feeds")
}

func TestFeedServiceClient_ListArticlesToCheck_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	articles := []*feedpb.ArticleToCheck{
		{
			ArticleId:        1,
			FeedId:           10,
			Url:              "https://example.com/article-1",
			PrevEtag:         "etag-1",
			PrevLastModified: "2024-01-01T00:00:00Z",
		},
		{
			ArticleId: 2,
			FeedId:    11,
			Url:       "https://example.com/article-2",
		},
	}

	mockClient := &MockFeedServiceClient{articles: articles, nextToken: "next"}
	client := &FeedServiceClient{client: mockClient, logger: logger}

	ctx := context.Background()
	window := models.ArticleCheckWindow{
		PublishedSince:    time.Now().Add(-24 * time.Hour),
		LastCheckedBefore: time.Now().Add(-1 * time.Hour),
	}

	page, err := client.ListArticlesToCheck(ctx, window, 10, "")

	require.NoError(t, err)
	require.NotNil(t, page)
	assert.Len(t, page.Items, 2)
	assert.Equal(t, "next", page.NextPageToken)
	assert.Equal(t, uint(1), page.Items[0].ArticleID)
	assert.Equal(t, "etag-1", page.Items[0].PrevETag)
}

func TestFeedServiceClient_ListArticlesToCheck_Error(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	mockClient := &MockFeedServiceClient{
		err: status.Error(codes.Internal, "boom"),
	}
	client := &FeedServiceClient{client: mockClient, logger: logger}

	ctx := context.Background()
	window := models.ArticleCheckWindow{PublishedSince: time.Now(), LastCheckedBefore: time.Now()}

	page, err := client.ListArticlesToCheck(ctx, window, 5, "token")

	require.Error(t, err)
	assert.Nil(t, page)
	assert.Contains(t, err.Error(), "failed to list articles to check")
}

func TestFeedServiceClient_ListArticlesToCheck_InvalidPageSize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockClient := &MockFeedServiceClient{}
	client := &FeedServiceClient{client: mockClient, logger: logger}

	ctx := context.Background()
	window := models.ArticleCheckWindow{PublishedSince: time.Now(), LastCheckedBefore: time.Now()}

	page, err := client.ListArticlesToCheck(ctx, window, 0, "")

	require.Error(t, err)
	assert.Nil(t, page)
	assert.Contains(t, err.Error(), "page size must be positive")
}
