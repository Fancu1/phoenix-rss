package ierr

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/pkg/logger"
)

// ErrorResponse represent the JSON structure returned to clients for errors
type ErrorResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// ErrorHandlerMiddleware create a middleware that handles errors in a centralized way
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// Get contextual logger
		log := logger.FromContext(c.Request.Context())

		// Process all errors, only respond with the first one
		var appErr *AppError
		var firstError error
		var foundAppError bool

		for _, ginErr := range c.Errors {
			if firstError == nil {
				firstError = ginErr.Err
			}

			// extract AppError from the error chain
			if errors.As(ginErr.Err, &appErr) {
				foundAppError = true
				break
			}

			// check for specific predefined errors
			if !foundAppError {
				appErr = findAppErrorByIs(ginErr.Err)
				if appErr != nil {
					foundAppError = true
					break
				}
			}
		}

		requestID, _ := logger.GetRequestID(c.Request.Context())

		if foundAppError && appErr != nil {
			// Handle known application error
			log.Warn("application error occurred",
				"error_code", appErr.Code,
				"error_message", appErr.Message,
				"http_status", appErr.HTTPStatus,
				"full_context", firstError.Error(), // Log the full wrapped error for debugging
				"cause", appErr.cause,
			)

			// Return structured error response with request_id
			c.AbortWithStatusJSON(appErr.HTTPStatus, ErrorResponse{
				Code:      appErr.Code,
				Message:   appErr.Message,
				RequestID: requestID,
			})
		} else {
			// Handle unknown/unexpected error
			log.Error("unexpected error occurred",
				"error", firstError.Error(),
				"request_method", c.Request.Method,
				"request_path", c.Request.URL.Path,
			)

			// Return generic internal server error (don't expose internal details)
			c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
				Code:      ErrInternalServer.Code,
				Message:   ErrInternalServer.Message,
				RequestID: requestID,
			})
		}
	}
}

// findAppErrorByIs check if the error chain contains any of our predefined AppErrors
func findAppErrorByIs(err error) *AppError {
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

	// check each predefined error
	for _, predefinedErr := range predefinedErrors {
		if errors.Is(err, predefinedErr) {
			return predefinedErr
		}
	}

	return nil
}

// AbortWithError records an error and aborts the request
func AbortWithError(c *gin.Context, err error) {
	c.Error(err)
	c.Abort()
}

// AbortWithAppError records an AppError and aborts the request
func AbortWithAppError(c *gin.Context, appErr *AppError) {
	c.Error(appErr)
	c.Abort()
}

// NewInternalError create a new internal server error with cause for logging
func NewInternalError(cause error) *AppError {
	return ErrInternalServer.WithCause(cause)
}

// NewDatabaseError create a new database error with cause for logging
func NewDatabaseError(cause error) *AppError {
	return ErrDatabaseError.WithCause(cause)
}

// NewTaskQueueError create a new task queue error with cause for logging
func NewTaskQueueError(cause error) *AppError {
	return ErrTaskQueueError.WithCause(cause)
}
