package ierr

import (
	"fmt"
	"net/http"
)

// AppError represents a structured application error with HTTP status code,
// internal error code, and user-friendly message
type AppError struct {
	Code       int    // Internal error code for API consumers
	Message    string // User-friendly error message
	HTTPStatus int    // HTTP status code to return
	cause      error  // Internal cause (for logging, not exposed to user)
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

// Unwrap implements the unwrapper interface for Go 1.13+ error handling
func (e *AppError) Unwrap() error {
	return e.cause
}

// WithCause returns a new AppError with the given cause wrapped
func (e *AppError) WithCause(cause error) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    e.Message,
		HTTPStatus: e.HTTPStatus,
		cause:      cause,
	}
}

// Predefined application errors
var (
	// User-related errors (1000-1099)
	ErrUserExists         = &AppError{Code: 1001, Message: "Username already exists", HTTPStatus: http.StatusConflict}
	ErrInvalidCredentials = &AppError{Code: 1002, Message: "Invalid credentials", HTTPStatus: http.StatusUnauthorized}
	ErrUserNotFound       = &AppError{Code: 1003, Message: "User not found", HTTPStatus: http.StatusNotFound}
	ErrInvalidToken       = &AppError{Code: 1004, Message: "Invalid or expired token", HTTPStatus: http.StatusUnauthorized}

	// Feed-related errors (1100-1199)
	ErrFeedNotFound      = &AppError{Code: 1101, Message: "Feed not found", HTTPStatus: http.StatusNotFound}
	ErrFeedAlreadyExists = &AppError{Code: 1102, Message: "Feed already exists", HTTPStatus: http.StatusConflict}
	ErrInvalidFeedURL    = &AppError{Code: 1103, Message: "Invalid feed URL", HTTPStatus: http.StatusBadRequest}
	ErrFeedFetchFailed   = &AppError{Code: 1104, Message: "Failed to fetch feed", HTTPStatus: http.StatusBadGateway}
	ErrNotSubscribed     = &AppError{Code: 1105, Message: "Not subscribed to this feed", HTTPStatus: http.StatusForbidden}
	ErrAlreadySubscribed = &AppError{Code: 1106, Message: "Already subscribed to this feed", HTTPStatus: http.StatusConflict}

	// Article-related errors (1200-1299)
	ErrArticleNotFound = &AppError{Code: 1201, Message: "Article not found", HTTPStatus: http.StatusNotFound}

	// Validation errors (1300-1399)
	ErrInvalidInput  = &AppError{Code: 1301, Message: "Invalid input", HTTPStatus: http.StatusBadRequest}
	ErrMissingField  = &AppError{Code: 1302, Message: "Required field is missing", HTTPStatus: http.StatusBadRequest}
	ErrInvalidFeedID = &AppError{Code: 1303, Message: "Invalid feed ID", HTTPStatus: http.StatusBadRequest}

	// Authorization errors (1400-1499)
	ErrUnauthorized = &AppError{Code: 1401, Message: "Authentication required", HTTPStatus: http.StatusUnauthorized}
	ErrForbidden    = &AppError{Code: 1402, Message: "Access denied", HTTPStatus: http.StatusForbidden}

	// System errors (9000+)
	ErrInternalServer = &AppError{Code: 9001, Message: "Internal server error", HTTPStatus: http.StatusInternalServerError}
	ErrDatabaseError  = &AppError{Code: 9002, Message: "Database error", HTTPStatus: http.StatusInternalServerError}
	ErrTaskQueueError = &AppError{Code: 9003, Message: "Task queue error", HTTPStatus: http.StatusInternalServerError}
)

// NewAppError creates a new AppError with the given parameters
func NewAppError(code int, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// NewValidationError creates a validation error with specific message
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:       1301,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}
