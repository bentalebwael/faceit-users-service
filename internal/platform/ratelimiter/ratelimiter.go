package ratelimiter

import (
	"github.com/bentalebwael/faceit-users-service/internal/config"
	"golang.org/x/time/rate"
)

// RateLimiter wraps the standard library rate limiter with our configuration
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewLimiter creates a new rate limiter with the specified configuration
func NewLimiter(cfg *config.Config) *RateLimiter {
	limit := rate.Limit(cfg.Rate.RequestsPerSecond)
	burst := cfg.Rate.Burst

	limiter := rate.NewLimiter(limit, burst)

	return &RateLimiter{
		limiter: limiter,
	}
}

// Allow returns true if a request should be allowed, false if it should be rejected
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}
