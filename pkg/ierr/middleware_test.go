package ierr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

func setupTestGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Add our error handler middleware
	r.Use(ErrorHandlerMiddleware())

	return r
}

func TestErrorHandlerMiddleware_NoError(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["message"])
}

func TestErrorHandlerMiddleware_AppError(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		// Add request context with logger for proper testing
		ctx := logger.WithRequestID(c.Request.Context(), "test-123")
		c.Request = c.Request.WithContext(ctx)

		c.Error(ErrUserExists)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, ErrUserExists.Code, response.Code)
	assert.Equal(t, ErrUserExists.Message, response.Message)
}

func TestErrorHandlerMiddleware_AppErrorWithCause(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		// Add request context with logger for proper testing
		ctx := logger.WithRequestID(c.Request.Context(), "test-123")
		c.Request = c.Request.WithContext(ctx)

		cause := errors.New("database connection failed")
		c.Error(ErrDatabaseError.WithCause(cause))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, ErrDatabaseError.Code, response.Code)
	assert.Equal(t, ErrDatabaseError.Message, response.Message)
}

func TestErrorHandlerMiddleware_WrappedError(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		// Add request context with logger for proper testing
		ctx := logger.WithRequestID(c.Request.Context(), "test-123")
		c.Request = c.Request.WithContext(ctx)

		// Test the new fmt.Errorf wrapped error handling
		wrappedErr := fmt.Errorf("user 'john_doe' already exists: %w", ErrUserExists)
		c.Error(wrappedErr)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, ErrUserExists.Code, response.Code)
	assert.Equal(t, ErrUserExists.Message, response.Message)
}

func TestErrorHandlerMiddleware_WrappedErrorWithContext(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		// Add request context with logger for proper testing
		ctx := logger.WithRequestID(c.Request.Context(), "test-123")
		c.Request = c.Request.WithContext(ctx)

		// Test deeply wrapped errors
		innerErr := fmt.Errorf("feed 123 not found: %w", ErrFeedNotFound)
		outerErr := fmt.Errorf("failed to fetch articles: %w", innerErr)
		c.Error(outerErr)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, ErrFeedNotFound.Code, response.Code)
	assert.Equal(t, ErrFeedNotFound.Message, response.Message)
}

func TestErrorHandlerMiddleware_UnknownError(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		// Add request context with logger for proper testing
		ctx := logger.WithRequestID(c.Request.Context(), "test-123")
		c.Request = c.Request.WithContext(ctx)

		// Simulate an unexpected error (not an AppError)
		c.Error(errors.New("unexpected database error"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should return generic internal server error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, ErrInternalServer.Code, response.Code)
	assert.Equal(t, ErrInternalServer.Message, response.Message)
}

func TestErrorHandlerMiddleware_MultipleErrors(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		// Add request context with logger for proper testing
		ctx := logger.WithRequestID(c.Request.Context(), "test-123")
		c.Request = c.Request.WithContext(ctx)

		// Add multiple errors - should handle the first AppError
		c.Error(errors.New("generic error"))
		c.Error(ErrInvalidCredentials)
		c.Error(ErrUserExists)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, ErrInvalidCredentials.Code, response.Code)
	assert.Equal(t, ErrInvalidCredentials.Message, response.Message)
}

func TestErrorHandlerMiddleware_MultipleWrappedErrors(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		// Add request context with logger for proper testing
		ctx := logger.WithRequestID(c.Request.Context(), "test-123")
		c.Request = c.Request.WithContext(ctx)

		// Add multiple wrapped errors - should handle the first one found
		c.Error(errors.New("generic error"))
		wrappedErr := fmt.Errorf("login failed for user 'test': %w", ErrInvalidCredentials)
		c.Error(wrappedErr)
		c.Error(ErrUserExists)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, ErrInvalidCredentials.Code, response.Code)
	assert.Equal(t, ErrInvalidCredentials.Message, response.Message)
}

func TestAbortWithError(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		// Add request context with logger
		ctx := logger.WithRequestID(c.Request.Context(), "test-123")
		c.Request = c.Request.WithContext(ctx)

		AbortWithError(c, ErrForbidden)
		// Early return to prevent further execution
		return
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, ErrForbidden.Code, response.Code)
	assert.Equal(t, ErrForbidden.Message, response.Message)
}

func TestAbortWithAppError(t *testing.T) {
	r := setupTestGin()

	r.GET("/test", func(c *gin.Context) {
		// Add request context with logger
		ctx := logger.WithRequestID(c.Request.Context(), "test-123")
		c.Request = c.Request.WithContext(ctx)

		AbortWithAppError(c, ErrNotSubscribed)
		// Early return to prevent further execution
		return
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, ErrNotSubscribed.Code, response.Code)
	assert.Equal(t, ErrNotSubscribed.Message, response.Message)
}

func TestFindAppErrorByIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected *AppError
	}{
		{
			name:     "direct AppError",
			err:      ErrUserExists,
			expected: ErrUserExists,
		},
		{
			name:     "wrapped AppError",
			err:      fmt.Errorf("user 'john' already exists: %w", ErrUserExists),
			expected: ErrUserExists,
		},
		{
			name:     "deeply wrapped AppError",
			err:      fmt.Errorf("registration failed: %w", fmt.Errorf("user 'john' already exists: %w", ErrUserExists)),
			expected: ErrUserExists,
		},
		{
			name:     "unknown error",
			err:      errors.New("unknown error"),
			expected: nil,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findAppErrorByIs(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
