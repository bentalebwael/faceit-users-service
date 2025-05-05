package interceptors

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bentalebwael/faceit-users-service/internal/platform/ratelimiter"
)

// UnaryRateLimitInterceptor returns a new unary server interceptor for rate limiting
func UnaryRateLimitInterceptor(limiter *ratelimiter.RateLimiter, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !limiter.Allow() {
			logger.Warn("rate limit exceeded",
				"method", info.FullMethod,
			)
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}
