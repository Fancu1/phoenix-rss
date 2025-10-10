package handler

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Fancu1/phoenix-rss/internal/events"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/internal/feed-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	article_eventspb "github.com/Fancu1/phoenix-rss/proto/gen/article_events"
	feedpb "github.com/Fancu1/phoenix-rss/protos/gen/go/feed"
)

type mockArticleService struct {
	mock.Mock
}

func (m *mockArticleService) FetchAndSaveArticles(ctx context.Context, feedID uint) ([]*models.Article, error) {
	args := m.Called(ctx, feedID)
	if v := args.Get(0); v != nil {
		return v.([]*models.Article), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockArticleService) ListArticlesByFeedID(ctx context.Context, userID, feedID uint) ([]*models.Article, error) {
	args := m.Called(ctx, userID, feedID)
	if v := args.Get(0); v != nil {
		return v.([]*models.Article), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockArticleService) GetArticleByID(ctx context.Context, userID, articleID uint) (*models.Article, error) {
	args := m.Called(ctx, userID, articleID)
	if v := args.Get(0); v != nil {
		return v.(*models.Article), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockArticleService) HandleArticleProcessed(ctx context.Context, event *article_eventspb.ArticleProcessedEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockArticleService) ListArticlesToCheck(ctx context.Context, publishedSince, lastCheckedBefore time.Time, pageSize int, pageToken string) ([]repository.ArticleCheckCandidate, string, error) {
	args := m.Called(ctx, publishedSince, lastCheckedBefore, pageSize, pageToken)
	var result []repository.ArticleCheckCandidate
	if v := args.Get(0); v != nil {
		result = v.([]repository.ArticleCheckCandidate)
	}
	return result, args.String(1), args.Error(2)
}

type noopFeedService struct{}

func (noopFeedService) AddFeedByURL(ctx context.Context, url string) (*models.Feed, error) {
	return nil, nil
}
func (noopFeedService) ListAllFeeds(ctx context.Context) ([]*models.Feed, error) { return nil, nil }
func (noopFeedService) SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error) {
	return nil, nil
}
func (noopFeedService) ListUserFeeds(ctx context.Context, userID uint) ([]*models.Feed, error) {
	return nil, nil
}
func (noopFeedService) UnsubscribeFromFeed(ctx context.Context, userID, feedID uint) error {
	return nil
}
func (noopFeedService) IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error) {
	return false, nil
}

func TestListArticlesToCheck_Success(t *testing.T) {
	mockArticles := new(mockArticleService)
	h := NewFeedServiceHandler(slogDiscard(), noopFeedService{}, mockArticles, events.Producer(nil))

	publishedSince := time.Now().Add(-24 * time.Hour).UTC()
	lastCheckedBefore := time.Now().Add(-4 * time.Hour).UTC()

	candidates := []repository.ArticleCheckCandidate{
		{ID: 1, FeedID: 2, URL: "https://example.com", HTTPETag: strPtr("etag"), HTTPLastModified: strPtr("2024-01-01T00:00:00Z")},
	}

	mockArticles.On("ListArticlesToCheck", mock.Anything, publishedSince, lastCheckedBefore, 25, "").Return(candidates, "next", nil)

	req := &feedpb.ListArticlesToCheckRequest{
		PublishedSince:    publishedSince.Format(time.RFC3339),
		LastCheckedBefore: lastCheckedBefore.Format(time.RFC3339),
		PageSize:          25,
	}

	resp, err := h.ListArticlesToCheck(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "next", resp.NextPageToken)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, uint64(1), resp.Items[0].ArticleId)
	assert.Equal(t, "etag", resp.Items[0].PrevEtag)

	mockArticles.AssertExpectations(t)
}

func TestListArticlesToCheck_InvalidArguments(t *testing.T) {
	mockArticles := new(mockArticleService)
	h := NewFeedServiceHandler(slogDiscard(), noopFeedService{}, mockArticles, events.Producer(nil))

	req := &feedpb.ListArticlesToCheckRequest{}
	_, err := h.ListArticlesToCheck(context.Background(), req)
	require.Error(t, err)
}

func TestListArticlesToCheck_ServiceError(t *testing.T) {
	mockArticles := new(mockArticleService)
	h := NewFeedServiceHandler(slogDiscard(), noopFeedService{}, mockArticles, events.Producer(nil))

	publishedSince := time.Now().Add(-24 * time.Hour).UTC()
	lastCheckedBefore := time.Now().Add(-4 * time.Hour).UTC()

	mockArticles.On("ListArticlesToCheck", mock.Anything, publishedSince, lastCheckedBefore, 500, "").Return(nil, "", ierr.NewDatabaseError(assert.AnError))

	req := &feedpb.ListArticlesToCheckRequest{
		PublishedSince:    publishedSince.Format(time.RFC3339),
		LastCheckedBefore: lastCheckedBefore.Format(time.RFC3339),
	}

	_, err := h.ListArticlesToCheck(context.Background(), req)
	require.Error(t, err)

	mockArticles.AssertExpectations(t)
}

func slogDiscard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func strPtr(s string) *string { return &s }
