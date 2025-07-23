package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestFromContext(t *testing.T) {
	// Test with nil context
	logger := FromContext(nil)
	if logger == nil {
		t.Error("Expected non-nil logger from nil context")
	}

	// Test with empty context
	ctx := context.Background()
	logger = FromContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger from empty context")
	}

	// Test with request ID
	ctx = WithRequestID(context.Background(), "test-request-123")
	logger = FromContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger from context with request ID")
	}

	// Test with task ID
	ctx = WithTaskID(context.Background(), "test-task-456")
	logger = FromContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger from context with task ID")
	}

	// Test with user ID
	ctx = WithUserID(context.Background(), 42)
	logger = FromContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger from context with user ID")
	}

	// Test with multiple context values
	ctx = WithRequestID(context.Background(), "test-request-123")
	ctx = WithTaskID(ctx, "test-task-456")
	ctx = WithUserID(ctx, 42)
	logger = FromContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger from context with multiple values")
	}
}

func TestContextKeys(t *testing.T) {
	// Test request ID
	ctx := WithRequestID(context.Background(), "test-request-123")
	requestID, ok := GetRequestID(ctx)
	if !ok || requestID != "test-request-123" {
		t.Errorf("Expected request ID 'test-request-123', got '%s', ok=%v", requestID, ok)
	}

	// Test task ID
	ctx = WithTaskID(context.Background(), "test-task-456")
	taskID, ok := GetTaskID(ctx)
	if !ok || taskID != "test-task-456" {
		t.Errorf("Expected task ID 'test-task-456', got '%s', ok=%v", taskID, ok)
	}

	// Test user ID
	ctx = WithUserID(context.Background(), 42)
	userID, ok := GetUserID(ctx)
	if !ok || userID != 42 {
		t.Errorf("Expected user ID 42, got %d, ok=%v", userID, ok)
	}

	// Test with value
	ctx = WithValue(context.Background(), "test_key", "test_value")
	value, ok := GetValue(ctx, "test_key")
	if !ok || value != "test_value" {
		t.Errorf("Expected value 'test_value', got '%v', ok=%v", value, ok)
	}
}

func TestWithContext(t *testing.T) {
	ctx := WithRequestID(context.Background(), "test-request-123")
	logger := WithContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger from WithContext")
	}
}

func TestNew(t *testing.T) {
	logger := New(slog.LevelDebug)
	if logger == nil {
		t.Error("Expected non-nil logger from New")
	}

	logger = New(slog.LevelInfo)
	if logger == nil {
		t.Error("Expected non-nil logger from New with Info level")
	}
}
