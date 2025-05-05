package ratelimiter

import (
	"testing"
	"time"

	"github.com/bentalebwael/faceit-users-service/internal/config"
)

func TestNewLimiter(t *testing.T) {
	tests := []struct {
		name  string
		rps   int
		burst int
	}{
		{
			name:  "default config",
			rps:   10,
			burst: 20,
		},
		{
			name:  "high throughput",
			rps:   1000,
			burst: 2000,
		},
		{
			name:  "low throughput",
			rps:   1,
			burst: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Rate: config.RateConfig{
					RequestsPerSecond: tt.rps,
					Burst:             tt.burst,
				},
			}

			limiter := NewLimiter(cfg)
			if limiter == nil {
				t.Error("NewLimiter() returned nil")
				return
			}

			if limiter.limiter == nil {
				t.Error("NewLimiter() internal limiter is nil")
			}
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	tests := []struct {
		name         string
		rps          int
		burst        int
		requests     int
		delay        time.Duration
		wantAllowed  int
		wantRejected int
	}{
		{
			name:         "allow within limit",
			rps:          10,
			burst:        1,
			requests:     5,
			delay:        time.Millisecond * 100,
			wantAllowed:  5,
			wantRejected: 0,
		},
		{
			name:         "reject over limit",
			rps:          1,
			burst:        1,
			requests:     10,
			delay:        time.Millisecond,
			wantAllowed:  1,
			wantRejected: 9,
		},
		{
			name:         "use burst capacity",
			rps:          1,
			burst:        5,
			requests:     5,
			delay:        0,
			wantAllowed:  5,
			wantRejected: 0,
		},
		{
			name:         "exceed burst capacity",
			rps:          1,
			burst:        5,
			requests:     10,
			delay:        0,
			wantAllowed:  5,
			wantRejected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Rate: config.RateConfig{
					RequestsPerSecond: tt.rps,
					Burst:             tt.burst,
				},
			}

			limiter := NewLimiter(cfg)

			allowed := 0
			rejected := 0

			for i := 0; i < tt.requests; i++ {
				if limiter.Allow() {
					allowed++
				} else {
					rejected++
				}

				if tt.delay > 0 {
					time.Sleep(tt.delay)
				}
			}

			if allowed != tt.wantAllowed {
				t.Errorf("Got %d allowed requests, want %d", allowed, tt.wantAllowed)
			}
			if rejected != tt.wantRejected {
				t.Errorf("Got %d rejected requests, want %d", rejected, tt.wantRejected)
			}
		})
	}
}

func TestRateLimiter_RateRecovery(t *testing.T) {
	cfg := &config.Config{
		Rate: config.RateConfig{
			RequestsPerSecond: 2, // 2 requests per second
			Burst:             1, // No burst capacity
		},
	}

	limiter := NewLimiter(cfg)

	if !limiter.Allow() {
		t.Error("First request was not allowed")
	}

	if limiter.Allow() {
		t.Error("Second immediate request was allowed when it should have been rejected")
	}

	time.Sleep(time.Second / 2)

	if !limiter.Allow() {
		t.Error("Request after waiting was not allowed")
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	cfg := &config.Config{
		Rate: config.RateConfig{
			RequestsPerSecond: 100,
			Burst:             10,
		},
	}

	limiter := NewLimiter(cfg)
	numGoroutines := 10
	requestsPerGoroutine := 20
	done := make(chan bool)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < requestsPerGoroutine; j++ {
				limiter.Allow()
				time.Sleep(time.Millisecond)
			}
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
