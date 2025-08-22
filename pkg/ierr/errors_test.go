package ierr

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appErr   *AppError
		expected string
	}{
		{
			name:     "error without cause",
			appErr:   ErrUserExists,
			expected: "Username already exists",
		},
		{
			name:     "error with cause",
			appErr:   ErrUserExists.WithCause(errors.New("database connection failed")),
			expected: "Username already exists: database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.appErr.Error())
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	appErr := ErrUserExists.WithCause(cause)

	assert.Equal(t, cause, appErr.Unwrap())
	assert.True(t, errors.Is(appErr, cause))
}

func TestAppError_WithCause(t *testing.T) {
	cause := errors.New("database error")
	appErr := ErrInvalidCredentials.WithCause(cause)

	assert.Equal(t, ErrInvalidCredentials.Code, appErr.Code)
	assert.Equal(t, ErrInvalidCredentials.Message, appErr.Message)
	assert.Equal(t, ErrInvalidCredentials.HTTPStatus, appErr.HTTPStatus)
	assert.Equal(t, cause, appErr.cause)
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name       string
		appErr     *AppError
		expectCode int
		expectHTTP int
	}{
		{"ErrUserExists", ErrUserExists, 1001, http.StatusConflict},
		{"ErrInvalidCredentials", ErrInvalidCredentials, 1002, http.StatusUnauthorized},
		{"ErrUserNotFound", ErrUserNotFound, 1003, http.StatusNotFound},
		{"ErrInvalidToken", ErrInvalidToken, 1004, http.StatusUnauthorized},
		{"ErrFeedNotFound", ErrFeedNotFound, 1101, http.StatusNotFound},
		{"ErrInvalidFeedURL", ErrInvalidFeedURL, 1103, http.StatusBadRequest},
		{"ErrNotSubscribed", ErrNotSubscribed, 1105, http.StatusForbidden},
		{"ErrInvalidInput", ErrInvalidInput, 1301, http.StatusBadRequest},
		{"ErrUnauthorized", ErrUnauthorized, 1401, http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, 1402, http.StatusForbidden},
		{"ErrInternalServer", ErrInternalServer, 9001, http.StatusInternalServerError},
		{"ErrDatabaseError", ErrDatabaseError, 9002, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectCode, tt.appErr.Code)
			assert.Equal(t, tt.expectHTTP, tt.appErr.HTTPStatus)
			assert.NotEmpty(t, tt.appErr.Message)
		})
	}
}

func TestNewAppError(t *testing.T) {
	code := 2001
	message := "Custom error"
	httpStatus := http.StatusTeapot

	appErr := NewAppError(code, message, httpStatus)

	assert.Equal(t, code, appErr.Code)
	assert.Equal(t, message, appErr.Message)
	assert.Equal(t, httpStatus, appErr.HTTPStatus)
	assert.Nil(t, appErr.cause)
}

func TestNewValidationError(t *testing.T) {
	message := "Validation failed"
	appErr := NewValidationError(message)

	assert.Equal(t, 1301, appErr.Code)
	assert.Equal(t, message, appErr.Message)
	assert.Equal(t, http.StatusBadRequest, appErr.HTTPStatus)
}

func TestErrorWrapping(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := ErrDatabaseError.WithCause(originalErr)

	// Test that errors.Is works
	assert.True(t, errors.Is(appErr, originalErr))

	// Test that errors.As works
	var targetAppErr *AppError
	require.True(t, errors.As(appErr, &targetAppErr))
	assert.Equal(t, ErrDatabaseError.Code, targetAppErr.Code)
	assert.Equal(t, originalErr, targetAppErr.cause)
}

func TestNewHelperFunctions(t *testing.T) {
	cause := errors.New("test cause")

	tests := []struct {
		name     string
		fn       func(error) *AppError
		expected *AppError
	}{
		{"NewInternalError", NewInternalError, ErrInternalServer},
		{"NewDatabaseError", NewDatabaseError, ErrDatabaseError},
		{"NewTaskQueueError", NewTaskQueueError, ErrTaskQueueError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(cause)
			assert.Equal(t, tt.expected.Code, result.Code)
			assert.Equal(t, tt.expected.Message, result.Message)
			assert.Equal(t, tt.expected.HTTPStatus, result.HTTPStatus)
			assert.Equal(t, cause, result.cause)
		})
	}
}
