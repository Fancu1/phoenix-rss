package logger

import (
	"context"
	"log/slog"
	"os"
)

// New creates a new slog.Logger with the specified level
// This function maintains backward compatibility with existing code
func New(level slog.Level) *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

// NewWithHandler creates a new slog.Logger with a custom handler
func NewWithHandler(handler slog.Handler) *slog.Logger {
	return slog.New(handler)
}

// FromContext creates a logger that automatically includes context values
// This is the recommended way to get a logger when you have a context
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		// Return a default logger if context is nil
		return New(slog.LevelInfo)
	}

	// Start with a base logger
	baseLogger := New(slog.LevelDebug)

	var args []any

	// Add request ID if present
	if requestID, ok := GetRequestID(ctx); ok {
		args = append(args, "request_id", requestID)
	}

	// Add task ID if present
	if taskID, ok := GetTaskID(ctx); ok {
		args = append(args, "task_id", taskID)
	}

	// Add user ID if present for audit logging
	if userID, ok := GetUserID(ctx); ok {
		args = append(args, "user_id", userID)
	}

	// If no context values found, return base logger
	if len(args) == 0 {
		return baseLogger
	}

	// Return logger with context attributes pre-populated
	return baseLogger.With(args...)
}

// WithContext returns a new logger that includes all context values
// This is an alias for FromContext for better API discoverability
func WithContext(ctx context.Context) *slog.Logger {
	return FromContext(ctx)
}

// Must wraps a logger creation function and panics on error
// Useful for initialization where logger creation failure is fatal
func Must(logger *slog.Logger, err error) *slog.Logger {
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	return logger
}
