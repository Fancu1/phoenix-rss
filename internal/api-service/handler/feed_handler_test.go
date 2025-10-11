package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/Fancu1/phoenix-rss/internal/feed-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

type stubFeedService struct {
	listUserFeedsFn        func(ctx context.Context, userID uint) ([]*models.Feed, error)
	subscribeToFeedFn      func(ctx context.Context, userID uint, url string) (*models.Feed, error)
	unsubscribeFromFeedFn  func(ctx context.Context, userID, feedID uint) error
	listAllFeedsFn         func(ctx context.Context) ([]*models.Feed, error)
	isUserSubscribedFn     func(ctx context.Context, userID, feedID uint) (bool, error)
	listUserFeedsCalls     int
	subscribeToFeedCalls   int
	unsubscribeFromCalls   int
}

func (s *stubFeedService) ListAllFeeds(ctx context.Context) ([]*models.Feed, error) {
	if s.listAllFeedsFn != nil {
		return s.listAllFeedsFn(ctx)
	}
	return nil, errors.New("not implemented")
}

func (s *stubFeedService) SubscribeToFeed(ctx context.Context, userID uint, url string) (*models.Feed, error) {
	s.subscribeToFeedCalls++
	if s.subscribeToFeedFn != nil {
		return s.subscribeToFeedFn(ctx, userID, url)
	}
	return nil, errors.New("not implemented")
}

func (s *stubFeedService) ListUserFeeds(ctx context.Context, userID uint) ([]*models.Feed, error) {
	s.listUserFeedsCalls++
	if s.listUserFeedsFn != nil {
		return s.listUserFeedsFn(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (s *stubFeedService) UnsubscribeFromFeed(ctx context.Context, userID, feedID uint) error {
	s.unsubscribeFromCalls++
	if s.unsubscribeFromFeedFn != nil {
		return s.unsubscribeFromFeedFn(ctx, userID, feedID)
	}
	return errors.New("not implemented")
}

func (s *stubFeedService) IsUserSubscribed(ctx context.Context, userID, feedID uint) (bool, error) {
	if s.isUserSubscribedFn != nil {
		return s.isUserSubscribedFn(ctx, userID, feedID)
	}
	return false, errors.New("not implemented")
}

func newTestContext(method, path string, body io.Reader) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, body)
	ctx.Request = req
	return ctx, w
}

func attachUserContext(c *gin.Context, userID uint) {
	baseCtx := logger.WithRequestID(context.Background(), "test-request")
	baseCtx = logger.WithUserID(baseCtx, userID)
	c.Request = c.Request.WithContext(baseCtx)
	c.Set("userID", userID)
}

func redisClientForTest(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return redisClient, mr
}

func cacheKey(userID uint) string {
	return "user:" + strconv.FormatUint(uint64(userID), 10) + ":feeds"
}

func TestFeedHandler_ListFeeds_UsesCacheWhenAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const userID = uint(42)
	feeds := []*models.Feed{
		{ID: 1, Title: "Cached Feed", URL: "https://example.com/rss"},
	}

	redisClient, mr := redisClientForTest(t)
	defer mr.Close()
	defer redisClient.Close()

	payload, err := json.Marshal(feeds)
	require.NoError(t, err)
	require.NoError(t, redisClient.Set(context.Background(), cacheKey(userID), payload, time.Minute).Err())

	stub := &stubFeedService{
		listUserFeedsFn: func(ctx context.Context, userID uint) ([]*models.Feed, error) {
			t.Fatalf("unexpected call to ListUserFeeds")
			return nil, nil
		},
	}

	handler := NewFeedHandler(stub, redisClient)

	ctx, w := newTestContext(http.MethodGet, "/api/v1/feeds", nil)
	attachUserContext(ctx, userID)

	handler.ListFeeds(ctx)

	require.Equal(t, http.StatusOK, w.Code)

	var resp []*models.Feed
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, feeds, resp)
	require.Equal(t, 0, stub.listUserFeedsCalls)
}

func TestFeedHandler_ListFeeds_PopulatesCacheOnMiss(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const userID = uint(77)
	feeds := []*models.Feed{
		{ID: 2, Title: "Fresh Feed", URL: "https://fresh.example.com/rss"},
	}

	redisClient, mr := redisClientForTest(t)
	defer mr.Close()
	defer redisClient.Close()

	stub := &stubFeedService{
		listUserFeedsFn: func(ctx context.Context, u uint) ([]*models.Feed, error) {
			require.Equal(t, userID, u)
			return feeds, nil
		},
	}

	handler := NewFeedHandler(stub, redisClient)

	ctx, w := newTestContext(http.MethodGet, "/api/v1/feeds", nil)
	attachUserContext(ctx, userID)

	handler.ListFeeds(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, stub.listUserFeedsCalls)

	var resp []*models.Feed
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, feeds, resp)

	cached, err := redisClient.Get(context.Background(), cacheKey(userID)).Result()
	require.NoError(t, err)

	var cachedFeeds []*models.Feed
	require.NoError(t, json.Unmarshal([]byte(cached), &cachedFeeds))
	require.Equal(t, feeds, cachedFeeds)

	ttl := mr.TTL(cacheKey(userID))
	require.True(t, ttl > 0 && ttl <= userFeedsCacheTTL)
}

func TestFeedHandler_ListFeeds_FallbackOnCacheError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const userID = uint(88)
	feeds := []*models.Feed{
		{ID: 3, Title: "Fallback Feed", URL: "https://fallback.example.com/rss"},
	}

	redisClient, mr := redisClientForTest(t)
	defer redisClient.Close()

	// Force Redis errors by shutting down the in-memory server.
	mr.Close()

	stub := &stubFeedService{
		listUserFeedsFn: func(ctx context.Context, u uint) ([]*models.Feed, error) {
			require.Equal(t, userID, u)
			return feeds, nil
		},
	}

	handler := NewFeedHandler(stub, redisClient)

	ctx, w := newTestContext(http.MethodGet, "/api/v1/feeds", nil)
	attachUserContext(ctx, userID)

	handler.ListFeeds(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, stub.listUserFeedsCalls)

	var resp []*models.Feed
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, feeds, resp)
}

func TestFeedHandler_ListFeeds_BadCacheContentFallsBack(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const userID = uint(101)
	feeds := []*models.Feed{
		{ID: 4, Title: "After Decode Fail", URL: "https://decode.example.com/rss"},
	}

	redisClient, mr := redisClientForTest(t)
	defer mr.Close()
	defer redisClient.Close()

	require.NoError(t, redisClient.Set(context.Background(), cacheKey(userID), []byte("not-json"), time.Minute).Err())

	stub := &stubFeedService{
		listUserFeedsFn: func(ctx context.Context, u uint) ([]*models.Feed, error) {
			require.Equal(t, userID, u)
			return feeds, nil
		},
	}

	handler := NewFeedHandler(stub, redisClient)

	ctx, w := newTestContext(http.MethodGet, "/api/v1/feeds", nil)
	attachUserContext(ctx, userID)

	handler.ListFeeds(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, stub.listUserFeedsCalls)

	var resp []*models.Feed
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, feeds, resp)
}

func TestFeedHandler_AddFeed_InvalidateCache(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		userID = uint(55)
	)

	redisClient, mr := redisClientForTest(t)
	defer mr.Close()
	defer redisClient.Close()

	key := cacheKey(userID)
	require.NoError(t, redisClient.Set(context.Background(), key, []byte("cached"), time.Minute).Err())

	newFeed := &models.Feed{ID: 9, Title: "New Feed", URL: "https://new.example.com/rss"}

	stub := &stubFeedService{
		subscribeToFeedFn: func(ctx context.Context, user uint, url string) (*models.Feed, error) {
			require.Equal(t, userID, user)
			require.Equal(t, newFeed.URL, url)
			return newFeed, nil
		},
	}

	handler := NewFeedHandler(stub, redisClient)

	ctx, w := newTestContext(http.MethodPost, "/api/v1/feeds", strings.NewReader(`{"url":"`+newFeed.URL+`"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	attachUserContext(ctx, userID)

	handler.AddFeed(ctx)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, 1, stub.subscribeToFeedCalls)

	_, err := redisClient.Get(context.Background(), key).Result()
	require.ErrorIs(t, err, redis.Nil)
}

func TestFeedHandler_UnsubscribeFeed_InvalidateCache(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		userID = uint(66)
		feedID = uint(77)
	)

	redisClient, mr := redisClientForTest(t)
	defer mr.Close()
	defer redisClient.Close()

	require.NoError(t, redisClient.Set(context.Background(), cacheKey(userID), []byte("cached"), time.Minute).Err())

	stub := &stubFeedService{
		unsubscribeFromFeedFn: func(ctx context.Context, user, feed uint) error {
			require.Equal(t, userID, user)
			require.Equal(t, feedID, feed)
			return nil
		},
	}

	handler := NewFeedHandler(stub, redisClient)

	ctx, w := newTestContext(http.MethodDelete, "/api/v1/feeds/"+strconv.Itoa(int(feedID)), nil)
	attachUserContext(ctx, userID)
	ctx.Params = gin.Params{
		{Key: "feed_id", Value: strconv.Itoa(int(feedID))},
	}

	handler.UnsubscribeFeed(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, stub.unsubscribeFromCalls)

	_, err := redisClient.Get(context.Background(), cacheKey(userID)).Result()
	require.ErrorIs(t, err, redis.Nil)
}

func TestFeedHandler_ListFeeds_NoCacheConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const userID = uint(91)
	feeds := []*models.Feed{
		{ID: 10, Title: "No Cache", URL: "https://nocache.example.com/rss"},
	}

	stub := &stubFeedService{
		listUserFeedsFn: func(ctx context.Context, u uint) ([]*models.Feed, error) {
			require.Equal(t, userID, u)
			return feeds, nil
		},
	}

	handler := NewFeedHandler(stub, nil)

	ctx, w := newTestContext(http.MethodGet, "/api/v1/feeds", nil)
	attachUserContext(ctx, userID)

	handler.ListFeeds(ctx)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, stub.listUserFeedsCalls)

	var resp []*models.Feed
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, feeds, resp)
}
