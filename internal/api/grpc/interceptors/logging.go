package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		method := info.FullMethod

		logger.Info("gRPC request started",
			"method", method,
			"request", fmt.Sprintf("%+v", req),
		)

		resp, err := handler(ctx, req)
		duration := time.Since(start)

		// Determine status code
		code := codes.OK
		if err != nil {
			if s, ok := status.FromError(err); ok {
				code = s.Code()
			} else {
				code = codes.Internal
			}
		}

		logger.Info("gRPC request completed",
			"method", method,
			"duration", duration,
			"status", code.String(),
		)

		return resp, err
	}
}
