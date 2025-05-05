package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/bentalebwael/faceit-users-service/internal/api"
	grpcapi "github.com/bentalebwael/faceit-users-service/internal/api/grpc"
	"github.com/bentalebwael/faceit-users-service/internal/api/grpc/interceptors"
	restapi "github.com/bentalebwael/faceit-users-service/internal/api/rest"
	"github.com/bentalebwael/faceit-users-service/internal/config"
	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
	"github.com/bentalebwael/faceit-users-service/internal/events"
	kafkaPlatform "github.com/bentalebwael/faceit-users-service/internal/platform/kafka"
	"github.com/bentalebwael/faceit-users-service/internal/platform/logger"
	"github.com/bentalebwael/faceit-users-service/internal/platform/postgres"
	"github.com/bentalebwael/faceit-users-service/internal/platform/ratelimiter"
	"github.com/bentalebwael/faceit-users-service/internal/platform/redis"
	"github.com/bentalebwael/faceit-users-service/internal/platform/tracer"
	"github.com/bentalebwael/faceit-users-service/internal/repository/cache"
	"github.com/bentalebwael/faceit-users-service/internal/repository/database"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.NewLogger(cfg)

	// Initialize tracer
	tp, err := tracer.NewTracerProvider(cfg)
	if err != nil {
		log.Error("failed to initialize tracer", "error", err)
		os.Exit(1) // Exit if tracer fails, might be critical
	}
	defer func() {
		log.Info("Shutting down Tracer...")
		if err := tracer.Shutdown(context.Background(), tp); err != nil {
			log.Error("failed to shutdown tracer", "error", err)
		}
		log.Info("Tracer shutdown complete.")
	}()
	log.Info("Tracer initialized")

	// Initialize Postgres connection
	db, err := postgres.NewConnection(cfg)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer postgres.Close(db)
	log.Info("Database connection established")

	// Initialize Redis client
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close(redisClient)
	log.Info("Redis connection established")

	// Initialize Kafka producer
	log.Info("Initialize Kafka producer...")
	kafkaWriter, err := kafkaPlatform.NewProducer(cfg, log) // Use alias
	if err != nil {
		log.Error("failed to create kafka producer", "error", err)
		os.Exit(1)
	}
	defer kafkaPlatform.Close(kafkaWriter) // Use alias
	log.Info("Kafka producer initialized")

	// Initialize rate limiter
	limiter := ratelimiter.NewLimiter(cfg)
	log.Info("Rate limiter initialized")

	// Initialize repositories and event publisher
	userRepo := database.NewUserRepository(db)
	cachedRepo := cache.NewCacheDecorator(userRepo, redisClient, &cfg.Redis)
	eventPublisher := events.NewUserEventPublisher(kafkaWriter)
	log.Info("Repositories and publisher initialized")

	// Initialize user service
	userService := user.NewService(cachedRepo, eventPublisher, log)
	log.Info("User service initialized")

	// Initialize health checker and run initial check
	healthChecker := api.NewHealthChecker(db, redisClient, kafkaWriter, log)
	if status := healthChecker.Check(context.Background()); status.Status == api.Unhealthy {
		log.Error("initial health check failed", "status", status, "error", err)
		os.Exit(1)
	}
	log.Info("Initial health check passed")

	// Initialize REST server
	httpServer := restapi.NewServer(cfg.API.Port, userService, healthChecker, limiter, log)
	log.Info("REST server initialized")

	// Initialize gRPC server with interceptors
	grpcOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptors.UnaryLoggingInterceptor(log)),
	}
	grpcServer := grpcapi.NewServer(cfg.GRPC.Port, userService, log, grpcOpts...)
	log.Info("gRPC server initialized")

	// Create error channel for server errors
	errChan := make(chan error, 1)

	// Start servers in goroutines
	go func() {
		log.Info("Starting HTTP server", "port", cfg.API.Port)
		if err := httpServer.Start(); err != nil {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	go func() {
		log.Info("Starting gRPC server", "port", cfg.GRPC.Port)
		if err := grpcServer.Start(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for error or shutdown signal
	select {
	case err := <-errChan:
		log.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-sigChan:
		log.Info("received shutdown signal", "signal", sig)
	}

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Graceful shutdown
	log.Info("Shutting down servers...")
	if err := httpServer.Stop(ctx); err != nil {
		log.Error("failed to stop HTTP server", "error", err)
	} else {
		log.Info("HTTP server stopped")
	}

	grpcServer.Stop()
	log.Info("gRPC server stopped")
	log.Info("Shutdown complete")
}
