package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

// RequestIDMiddleware adds a unique request ID to each HTTP request
// It checks for existing X-Request-ID header (from upstream services)
// or generates a new UUID if none exists
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in headers (from upstream)
		requestID := c.GetHeader("X-Request-ID")

		// Generate new UUID if no request ID provided
		if requestID == "" {
			requestID = uuid.New().String()[:8]
		}

		// Set request ID in Gin context for handlers to use
		c.Set("request_id", requestID)

		// Set request ID in response header for client/debugging
		c.Header("X-Request-ID", requestID)

		// Create enhanced context with request ID for downstream services
		ctx := logger.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetRequestIDFromContext extracts request ID from Gin context
// This is a helper function for handlers that need the request ID
func GetRequestIDFromContext(c *gin.Context) (string, bool) {
	requestID, exists := c.Get("request_id")
	if !exists {
		return "", false
	}

	id, ok := requestID.(string)
	return id, ok
}

type AuthMiddleware struct {
	userService core.UserServiceInterface
}

func NewAuthMiddleware(userService core.UserServiceInterface) *AuthMiddleware {
	return &AuthMiddleware{
		userService: userService,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Error(ierr.ErrUnauthorized.WithCause(fmt.Errorf("authorization header required")))
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Error(ierr.ErrUnauthorized.WithCause(fmt.Errorf("invalid authorization header format")))
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Validate token
		token, err := m.userService.ValidateToken(tokenString)
		if err != nil {
			c.Error(err) // Already wrapped as AppError in ValidateToken
			c.Abort()
			return
		}

		// Extract user ID from token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Error(ierr.ErrInvalidToken.WithCause(fmt.Errorf("invalid token claims")))
			c.Abort()
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.Error(ierr.ErrInvalidToken.WithCause(fmt.Errorf("invalid user_id in token")))
			c.Abort()
			return
		}

		userID := uint(userIDFloat)

		// Get user details
		user, err := m.userService.GetUserFromToken(tokenString)
		if err != nil {
			c.Error(err) // Already wrapped as AppError in GetUserFromToken
			c.Abort()
			return
		}

		// Set user context in Gin
		c.Set("userID", userID)
		c.Set("user", user)

		// Also add user ID to request context for audit logging
		ctx := logger.WithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// Helper function to get user ID from context
func GetUserIDFromContext(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, false
	}

	id, ok := userID.(uint)
	return id, ok
}
