package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	// defaultWriter is the writer used by all loggers, can be stdout or file+stdout
	defaultWriter io.Writer = os.Stdout
	writerMu      sync.RWMutex
	logFile       *os.File
)

// InitFromEnv initializes the logger based on LOG_FILE environment variable.
func InitFromEnv() error {
	logFilePath := os.Getenv("LOG_FILE")
	if logFilePath == "" {
		return nil // No file logging, use stdout only
	}

	return InitWithFile(logFilePath)
}

// InitWithFile configures the logger to write to both stdout and the specified file.
func InitWithFile(filePath string) error {
	writerMu.Lock()
	defer writerMu.Unlock()

	if logFile != nil {
		_ = logFile.Close()
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	logFile = f
	defaultWriter = io.MultiWriter(os.Stdout, f)
	return nil
}

// Close closes the log file if one was opened. Call this on application shutdown.
func Close() error {
	writerMu.Lock()
	defer writerMu.Unlock()

	if logFile != nil {
		err := logFile.Close()
		logFile = nil
		defaultWriter = os.Stdout
		return err
	}
	return nil
}

func getWriter() io.Writer {
	writerMu.RLock()
	defer writerMu.RUnlock()
	return defaultWriter
}

func New(level slog.Level) *slog.Logger {
	handler := slog.NewTextHandler(getWriter(), &slog.HandlerOptions{Level: level})
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
