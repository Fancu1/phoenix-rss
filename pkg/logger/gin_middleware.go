package logger

import (
	"time"

	"github.com/gin-gonic/gin"
)

// GinLoggingMiddleware returns a Gin middleware that logs HTTP requests.
// It records method, path, status, latency, client IP, and context values (request_id, user_id).
func GinLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency and get status
		latency := time.Since(start)
		status := c.Writer.Status()

		// Get contextual logger with request_id and user_id
		log := FromContext(c.Request.Context())

		// Build log attributes
		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"body_size", c.Writer.Size(),
		}

		if query != "" {
			attrs = append(attrs, "query", query)
		}

		if c.Request.UserAgent() != "" {
			attrs = append(attrs, "user_agent", c.Request.UserAgent())
		}

		// Log at appropriate level based on status code
		switch {
		case status >= 500:
			log.Error("HTTP request completed", attrs...)
		case status >= 400:
			log.Warn("HTTP request completed", attrs...)
		default:
			log.Info("HTTP request completed", attrs...)
		}
	}
}
