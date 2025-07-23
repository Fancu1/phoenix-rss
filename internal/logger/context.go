package logger

import (
	"context"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// RequestIDKey is the context key for HTTP request IDs
	RequestIDKey ContextKey = "request_id"

	// TaskIDKey is the context key for async task IDs
	TaskIDKey ContextKey = "task_id"

	// UserIDKey is the context key for authenticated user IDs
	UserIDKey ContextKey = "user_id"
)

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithTaskID adds a task ID to the context
func WithTaskID(ctx context.Context, taskID string) context.Context {
	return context.WithValue(ctx, TaskIDKey, taskID)
}

// WithUserID adds a user ID to the context for audit logging
func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithValue adds an arbitrary key-value pair to the context
// This is useful for adding additional context information for logging
func WithValue(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, ContextKey(key), value)
}

// GetRequestID extracts the request ID from context
func GetRequestID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	requestID, ok := ctx.Value(RequestIDKey).(string)
	return requestID, ok
}

// GetTaskID extracts the task ID from context
func GetTaskID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	taskID, ok := ctx.Value(TaskIDKey).(string)
	return taskID, ok
}

// GetUserID extracts the user ID from context
func GetUserID(ctx context.Context) (uint, bool) {
	if ctx == nil {
		return 0, false
	}
	userID, ok := ctx.Value(UserIDKey).(uint)
	return userID, ok
}

// GetValue extracts an arbitrary value from context
func GetValue(ctx context.Context, key string) (interface{}, bool) {
	if ctx == nil {
		return nil, false
	}
	value := ctx.Value(ContextKey(key))
	return value, value != nil
}
