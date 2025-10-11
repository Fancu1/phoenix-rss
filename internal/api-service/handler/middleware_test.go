package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/Fancu1/phoenix-rss/internal/user-service/models"
)

type stubUserService struct {
	validateTokenFn    func(tokenString string) (*jwt.Token, error)
	getUserFromTokenFn func(tokenString string) (*models.User, error)

	validateCalls int
	getUserCalls  int
}

func (s *stubUserService) Register(username, password string) (*models.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserService) Login(username, password string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *stubUserService) ValidateToken(tokenString string) (*jwt.Token, error) {
	s.validateCalls++
	return s.validateTokenFn(tokenString)
}

func (s *stubUserService) GetUserFromToken(tokenString string) (*models.User, error) {
	s.getUserCalls++
	return s.getUserFromTokenFn(tokenString)
}

func newStubUserService(userID uint, username string) *stubUserService {
	return &stubUserService{
		validateTokenFn: func(tokenString string) (*jwt.Token, error) {
			return &jwt.Token{
				Claims: jwt.MapClaims{
					"user_id": float64(userID),
				},
				Valid: true,
			}, nil
		},
		getUserFromTokenFn: func(tokenString string) (*models.User, error) {
			return &models.User{
				ID:       userID,
				Username: username,
			}, nil
		},
	}
}

func TestAuthMiddleware_CacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	const token = "cached-token"
	payload := cachedUser{ID: 101, Username: "cached-user"}
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, redisClient.Set(context.Background(), cacheKeyForToken(token), encoded, tokenCacheTTL).Err())

	userSvc := newStubUserService(999, "should-not-be-used")
	middleware := NewAuthMiddleware(userSvc, redisClient)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/feeds", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx.Request = req

	middleware.RequireAuth()(ctx)

	require.False(t, ctx.IsAborted())
	require.Equal(t, 0, userSvc.validateCalls)
	require.Equal(t, 0, userSvc.getUserCalls)

	userIDValue, exists := ctx.Get("userID")
	require.True(t, exists)
	require.Equal(t, payload.ID, userIDValue.(uint))

	userValue, exists := ctx.Get("user")
	require.True(t, exists)
	user := userValue.(*models.User)
	require.Equal(t, payload.Username, user.Username)
}

func TestAuthMiddleware_CacheMissStoresValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	const (
		token    = "new-token"
		userID   = uint(202)
		username = "new-user"
	)

	userSvc := newStubUserService(userID, username)
	middleware := NewAuthMiddleware(userSvc, redisClient)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/feeds", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx.Request = req

	middleware.RequireAuth()(ctx)

	require.False(t, ctx.IsAborted())
	require.Equal(t, 1, userSvc.validateCalls)
	require.Equal(t, 1, userSvc.getUserCalls)

	value, err := redisClient.Get(context.Background(), cacheKeyForToken(token)).Result()
	require.NoError(t, err)

	var cached cachedUser
	require.NoError(t, json.Unmarshal([]byte(value), &cached))
	require.Equal(t, userID, cached.ID)
	require.Equal(t, username, cached.Username)
	require.True(t, mr.TTL(cacheKeyForToken(token)) <= tokenCacheTTL)
}

func TestAuthMiddleware_RedisErrorsFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr, err := miniredis.Run()
	require.NoError(t, err)

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	// Close the fake redis to force network errors on GET/SET.
	mr.Close()

	const (
		token    = "error-token"
		userID   = uint(303)
		username = "error-user"
	)

	userSvc := newStubUserService(userID, username)
	middleware := NewAuthMiddleware(userSvc, redisClient)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/feeds", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx.Request = req

	middleware.RequireAuth()(ctx)

	require.False(t, ctx.IsAborted())
	require.Equal(t, 1, userSvc.validateCalls)
	require.Equal(t, 1, userSvc.getUserCalls)

	userIDValue, exists := ctx.Get("userID")
	require.True(t, exists)
	require.Equal(t, userID, userIDValue.(uint))
}

func TestCacheKeyForToken(t *testing.T) {
	require.Equal(t, tokenCacheKeyPrefix+"foo", cacheKeyForToken("foo"))
	require.Equal(t, tokenCacheKeyPrefix, cacheKeyForToken(""))
}
