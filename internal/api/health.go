package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

const (
	Healthy   = "healthy"
	Unhealthy = "unhealthy"
)

type HealthStatus struct {
	Status  string  `json:"status"`
	Details Details `json:"details"`
}

type Details struct {
	Database  string    `json:"database"`
	Redis     string    `json:"redis"`
	Kafka     string    `json:"kafka"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthChecker performs health checks on dependencies
type HealthChecker struct {
	db     *sqlx.DB
	redis  *redis.Client
	kafka  *kafka.Writer
	logger *slog.Logger
}

func NewHealthChecker(db *sqlx.DB, redis *redis.Client, kafka *kafka.Writer, logger *slog.Logger) *HealthChecker {
	return &HealthChecker{
		db:     db,
		redis:  redis,
		kafka:  kafka,
		logger: logger,
	}
}

// Check performs health checks on all dependencies
func (h *HealthChecker) Check(ctx context.Context) *HealthStatus {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	status := &HealthStatus{
		Details: Details{
			Timestamp: time.Now(),
		},
	}

	if err := h.db.PingContext(ctx); err != nil {
		h.logger.Error("database health check failed", "error", err)
		status.Details.Database = Unhealthy
	} else {
		status.Details.Database = Healthy
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		h.logger.Error("redis health check failed", "error", err)
		status.Details.Redis = Unhealthy
	} else {
		status.Details.Redis = Healthy
	}

	// Check Kafka by connecting directly to the broker
	dialer := &kafka.Dialer{
		Timeout:   3 * time.Second,
		DualStack: true,
	}
	brokerConn, err := dialer.DialContext(ctx, "tcp", h.kafka.Addr.String())
	if err != nil {
		h.logger.Error("kafka connection failed", "error", err)
		status.Details.Kafka = Unhealthy
	} else {
		defer brokerConn.Close()
		// Try to fetch broker metadata as a lightweight health check
		_, err := brokerConn.Brokers()
		if err != nil {
			h.logger.Error("kafka metadata fetch failed", "error", err)
			status.Details.Kafka = Unhealthy
		} else {
			status.Details.Kafka = Healthy
		}
	}

	if status.Details.Database == Healthy &&
		status.Details.Redis == Healthy &&
		status.Details.Kafka == Healthy {
		status.Status = Healthy
	} else {
		status.Status = Unhealthy
	}

	return status
}
