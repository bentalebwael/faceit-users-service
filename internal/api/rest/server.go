package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bentalebwael/faceit-users-service/internal/api"
	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
	"github.com/bentalebwael/faceit-users-service/internal/platform/ratelimiter"
)

type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	logger     *slog.Logger
}

func NewServer(port int, service *user.Service, healthChecker *api.HealthChecker, limiter *ratelimiter.RateLimiter, logger *slog.Logger) *Server {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	handler := NewHandler(service, healthChecker, logger)
	router := setupRouter(handler, limiter, logger)

	// Configure HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	return &Server{
		router:     router,
		httpServer: httpServer,
		logger:     logger,
	}
}

func (s *Server) Start() error {
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping HTTP server")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to stop HTTP server: %w", err)
	}

	return nil
}
