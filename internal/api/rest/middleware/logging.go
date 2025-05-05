package middleware

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a Gin middleware for request logging
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method

		logger.Info("HTTP request started",
			"method", method,
			"path", path,
			"query", query,
		)

		// Process request
		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		logger.Info("HTTP request completed",
			"method", method,
			"path", path,
			"status", status,
			"duration", fmt.Sprintf("%.2fs", duration.Seconds()),
		)
	}
}
