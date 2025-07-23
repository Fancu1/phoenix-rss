package ierr

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/internal/logger"
)

// ErrorResponse represents the JSON structure returned to clients for errors
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorHandlerMiddleware creates a middleware that handles errors in a centralized way
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Execute the handler chain first
		c.Next()

		// Check if there are any errors to handle
		if len(c.Errors) == 0 {
			return
		}

		// Get contextual logger for proper error logging
		log := logger.FromContext(c.Request.Context())

		// Process all errors, but only respond with the first one
		var appErr *AppError
		var firstError error
		var foundAppError bool

		for _, ginErr := range c.Errors {
			if firstError == nil {
				firstError = ginErr.Err
			}

			// Try to extract AppError from the error chain using errors.As
			if errors.As(ginErr.Err, &appErr) {
				foundAppError = true
				break
			}

			// Also try using errors.Is to check for specific predefined errors
			if !foundAppError {
				appErr = findAppErrorByIs(ginErr.Err)
				if appErr != nil {
					foundAppError = true
					break
				}
			}
		}

		if foundAppError && appErr != nil {
			// Handle known application error
			log.Warn("application error occurred",
				"error_code", appErr.Code,
				"error_message", appErr.Message,
				"http_status", appErr.HTTPStatus,
				"full_context", firstError.Error(), // Log the full wrapped error for debugging
				"cause", appErr.cause,
			)

			// Return structured error response using the AppError's predefined values
			c.JSON(appErr.HTTPStatus, ErrorResponse{
				Code:    appErr.Code,
				Message: appErr.Message,
			})
		} else {
			// Handle unknown/unexpected error
			log.Error("unexpected error occurred",
				"error", firstError.Error(),
				"request_method", c.Request.Method,
				"request_path", c.Request.URL.Path,
			)

			// Return generic internal server error (don't expose internal details)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Code:    ErrInternalServer.Code,
				Message: ErrInternalServer.Message,
			})
		}

		// Abort to prevent further processing
		c.Abort()
	}
}

// findAppErrorByIs checks if the error chain contains any of our predefined AppErrors
// using errors.Is, which works well with fmt.Errorf wrapped errors
func findAppErrorByIs(err error) *AppError {
	// Define a list of all our predefined errors for checking
	predefinedErrors := []*AppError{
		// User-related errors
		ErrUserExists,
		ErrInvalidCredentials,
		ErrUserNotFound,
		ErrInvalidToken,

		// Feed-related errors
		ErrFeedNotFound,
		ErrFeedAlreadyExists,
		ErrInvalidFeedURL,
		ErrFeedFetchFailed,
		ErrNotSubscribed,
		ErrAlreadySubscribed,

		// Article-related errors
		ErrArticleNotFound,

		// Validation errors
		ErrInvalidInput,
		ErrMissingField,
		ErrInvalidFeedID,

		// Authorization errors
		ErrUnauthorized,
		ErrForbidden,

		// System errors
		ErrInternalServer,
		ErrDatabaseError,
		ErrTaskQueueError,
	}

	// Check each predefined error using errors.Is
	for _, predefinedErr := range predefinedErrors {
		if errors.Is(err, predefinedErr) {
			return predefinedErr
		}
	}

	return nil
}

// AbortWithError is a helper function that records an error and aborts the request
// This is useful when you want to immediately stop processing and return an error
func AbortWithError(c *gin.Context, err error) {
	c.Error(err)
	c.Abort()
}

// AbortWithAppError is a helper function that records an AppError and aborts the request
func AbortWithAppError(c *gin.Context, appErr *AppError) {
	c.Error(appErr)
	c.Abort()
}

// NewInternalError creates a new internal server error with cause for logging
func NewInternalError(cause error) *AppError {
	return ErrInternalServer.WithCause(cause)
}

// NewDatabaseError creates a new database error with cause for logging
func NewDatabaseError(cause error) *AppError {
	return ErrDatabaseError.WithCause(cause)
}

// NewTaskQueueError creates a new task queue error with cause for logging
func NewTaskQueueError(cause error) *AppError {
	return ErrTaskQueueError.WithCause(cause)
}
