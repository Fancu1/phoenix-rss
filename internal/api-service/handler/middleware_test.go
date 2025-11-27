package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/Fancu1/phoenix-rss/internal/user-service/models"
)

const testJWTSecret = "test-secret-key"

func generateTestToken(t *testing.T, userID uint, username string, exp time.Time) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  float64(userID),
		"username": username,
		"exp":      exp.Unix(),
		"iat":      time.Now().Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		userID   = uint(101)
		username = "testuser"
	)
	token := generateTestToken(t, userID, username, time.Now().Add(time.Hour))

	middleware := NewAuthMiddleware(testJWTSecret)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/feeds", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx.Request = req

	middleware.RequireAuth()(ctx)

	require.False(t, ctx.IsAborted())

	userIDValue, exists := ctx.Get("userID")
	require.True(t, exists)
	require.Equal(t, userID, userIDValue.(uint))

	userValue, exists := ctx.Get("user")
	require.True(t, exists)
	user := userValue.(*models.User)
	require.Equal(t, username, user.Username)
	require.Equal(t, userID, user.ID)
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token := generateTestToken(t, 1, "expired", time.Now().Add(-time.Hour))
	middleware := NewAuthMiddleware(testJWTSecret)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/feeds", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx.Request = req

	middleware.RequireAuth()(ctx)

	require.True(t, ctx.IsAborted())
	require.Len(t, ctx.Errors, 1)
}

func TestAuthMiddleware_InvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  float64(1),
		"username": "test",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte("wrong-secret"))

	middleware := NewAuthMiddleware(testJWTSecret)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/feeds", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	ctx.Request = req

	middleware.RequireAuth()(ctx)

	require.True(t, ctx.IsAborted())
	require.Len(t, ctx.Errors, 1)
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewAuthMiddleware(testJWTSecret)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/feeds", nil)
	ctx.Request = req

	middleware.RequireAuth()(ctx)

	require.True(t, ctx.IsAborted())
	require.Len(t, ctx.Errors, 1)
}

func TestAuthMiddleware_InvalidAuthHeaderFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middleware := NewAuthMiddleware(testJWTSecret)

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "token-without-bearer"},
		{"wrong prefix", "Basic sometoken"},
		{"empty bearer", "Bearer "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest(http.MethodGet, "/feeds", nil)
			req.Header.Set("Authorization", tc.header)
			ctx.Request = req

			middleware.RequireAuth()(ctx)

			require.True(t, ctx.IsAborted())
		})
	}
}

func TestAuthMiddleware_MissingClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			name: "missing user_id",
			claims: jwt.MapClaims{
				"username": "test",
				"exp":      time.Now().Add(time.Hour).Unix(),
			},
		},
		{
			name: "missing username",
			claims: jwt.MapClaims{
				"user_id": float64(1),
				"exp":     time.Now().Add(time.Hour).Unix(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, tc.claims)
			signed, _ := token.SignedString([]byte(testJWTSecret))

			middleware := NewAuthMiddleware(testJWTSecret)

			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest(http.MethodGet, "/feeds", nil)
			req.Header.Set("Authorization", "Bearer "+signed)
			ctx.Request = req

			middleware.RequireAuth()(ctx)

			require.True(t, ctx.IsAborted())
			require.Len(t, ctx.Errors, 1)
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("generates new request ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		ctx.Request = req

		RequestIDMiddleware()(ctx)

		id, ok := GetRequestIDFromContext(ctx)
		require.True(t, ok)
		require.Len(t, id, 8)
		require.Equal(t, id, w.Header().Get("X-Request-ID"))
	})

	t.Run("propagates existing request ID", func(t *testing.T) {
		const existingID = "upstream1"

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", existingID)
		ctx.Request = req

		RequestIDMiddleware()(ctx)

		id, ok := GetRequestIDFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, existingID, id)
		require.Equal(t, existingID, w.Header().Get("X-Request-ID"))
	})
}
