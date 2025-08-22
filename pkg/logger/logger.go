package logger

import (
	"context"
	"log/slog"
	"os"
)

// New create a new slog.Logger with the specified level
func New(level slog.Level) *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

// NewWithHandler create a new slog.Logger with a custom handler
func NewWithHandler(handler slog.Handler) *slog.Logger {
	return slog.New(handler)
}

// FromContext create a logger that automatically includes context values
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return New(slog.LevelInfo)
	}

	baseLogger := New(slog.LevelDebug)

	var args []any

	if requestID, ok := GetRequestID(ctx); ok {
		args = append(args, "request_id", requestID)
	}

	if taskID, ok := GetTaskID(ctx); ok {
		args = append(args, "task_id", taskID)
	}

	if userID, ok := GetUserID(ctx); ok {
		args = append(args, "user_id", userID)
	}

	// If no context values found, return base logger
	if len(args) == 0 {
		return baseLogger
	}

	return baseLogger.With(args...)
}

// WithContext return a new logger that includes all context values
func WithContext(ctx context.Context) *slog.Logger {
	return FromContext(ctx)
}

// Must wrap a logger creation function and panics on error
func Must(logger *slog.Logger, err error) *slog.Logger {
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	return logger
}
