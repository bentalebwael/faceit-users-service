package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bentalebwael/faceit-users-service/internal/platform/ratelimiter"
)

// RateLimit returns a Gin middleware for request rate limiting
func RateLimit(limiter *ratelimiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    "rate_limit_exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
