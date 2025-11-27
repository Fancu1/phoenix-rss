package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Fancu1/phoenix-rss/internal/user-service/models"
	"github.com/Fancu1/phoenix-rss/pkg/ierr"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

// RequestIDMiddleware propagates or generates a request ID for distributed tracing.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()[:8]
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), requestID))

		c.Next()
	}
}

// GetRequestIDFromContext retrieves the request ID from context.
func GetRequestIDFromContext(c *gin.Context) (string, bool) {
	if v, ok := c.Get("request_id"); ok {
		return v.(string), true
	}
	return "", false
}

// AuthMiddleware validates JWT tokens locally using shared secret.
type AuthMiddleware struct {
	jwtSecret []byte
}

// NewAuthMiddleware creates an AuthMiddleware with the given secret.
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: []byte(jwtSecret)}
}

// RequireAuth enforces JWT authentication and populates user context.
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Error(ierr.ErrUnauthorized.WithCause(fmt.Errorf("authorization header required")))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Error(ierr.ErrUnauthorized.WithCause(fmt.Errorf("invalid authorization header format")))
			c.Abort()
			return
		}

		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.jwtSecret, nil
		})
		if err != nil {
			c.Error(ierr.ErrInvalidToken.WithCause(err))
			c.Abort()
			return
		}
		if !token.Valid {
			c.Error(ierr.ErrInvalidToken.WithCause(fmt.Errorf("token validation failed")))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Error(ierr.ErrInvalidToken.WithCause(fmt.Errorf("invalid token claims")))
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			c.Error(ierr.ErrInvalidToken.WithCause(fmt.Errorf("missing user_id claim")))
			c.Abort()
			return
		}

		username, ok := claims["username"].(string)
		if !ok {
			c.Error(ierr.ErrInvalidToken.WithCause(fmt.Errorf("missing username claim")))
			c.Abort()
			return
		}

		user := &models.User{ID: uint(userID), Username: username}
		c.Set("userID", user.ID)
		c.Set("user", user)
		c.Request = c.Request.WithContext(logger.WithUserID(c.Request.Context(), user.ID))

		c.Next()
	}
}

// GetUserIDFromContext retrieves the authenticated user ID from context.
func GetUserIDFromContext(c *gin.Context) (uint, bool) {
	if v, ok := c.Get("userID"); ok {
		return v.(uint), true
	}
	return 0, false
}
