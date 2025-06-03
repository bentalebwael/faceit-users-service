package api

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNetAddr implements net.Addr for testing Kafka writer address.
type mockNetAddr struct{ network, address string }

func (m mockNetAddr) Network() string { return m.network }
func (m mockNetAddr) String() string  { return m.address }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// setupMocks creates mocks for DB, Redis, and Kafka for testing.
// It returns the mocks and the HealthChecker instance.
func setupMocks(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock, *redis.Client, redismock.ClientMock, *kafka.Writer, *HealthChecker) {
	t.Helper()

	mockDb, dbMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDb, "sqlmock")

	redisClient, redisMock := redismock.NewClientMock()

	// Use a likely non-existent port to simulate connection failure for Kafka check.
	kafkaWriter := &kafka.Writer{
		Addr: mockNetAddr{network: "tcp", address: "localhost:99999"},
	}

	logger := discardLogger()

	hc := NewHealthChecker(sqlxDB, redisClient, kafkaWriter, logger)

	t.Cleanup(func() {
		mockDb.Close()
		redisClient.Close()
	})

	return sqlxDB, dbMock, redisClient, redisMock, kafkaWriter, hc
}

func TestNewHealthChecker(t *testing.T) {
	t.Parallel()
	sqlxDB, _, redisClient, _, kafkaWriter, _ := setupMocks(t)
	logger := discardLogger()

	hc := NewHealthChecker(sqlxDB, redisClient, kafkaWriter, logger)

	assert.NotNil(t, hc)
	assert.Equal(t, sqlxDB, hc.db)
	assert.Equal(t, redisClient, hc.redis)
	assert.Equal(t, kafkaWriter, hc.kafka)
	assert.Equal(t, logger, hc.logger)
}

func TestHealthChecker_Check_DBUnhealthy(t *testing.T) {
	t.Parallel()
	_, dbMock, _, redisMock, _, hc := setupMocks(t)

	dbErr := errors.New("db connection failed")
	dbMock.ExpectPing().WillReturnError(dbErr) // DB Unhealthy
	redisMock.ExpectPing().SetVal("PONG")      // Redis Healthy

	status := hc.Check(context.Background())

	assert.Equal(t, Unhealthy, status.Status)
	assert.Equal(t, Unhealthy, status.Details.Database)
	assert.Equal(t, Healthy, status.Details.Redis)
	assert.Equal(t, Unhealthy, status.Details.Kafka) // Kafka check will fail
	assert.WithinDuration(t, time.Now(), status.Details.Timestamp, 1*time.Second)

	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestHealthChecker_Check_RedisUnhealthy(t *testing.T) {
	t.Parallel()
	_, dbMock, _, redisMock, _, hc := setupMocks(t)

	redisErr := errors.New("redis connection refused")
	dbMock.ExpectPing().WillReturnError(nil) // DB Healthy
	redisMock.ExpectPing().SetErr(redisErr)  // Redis Unhealthy

	status := hc.Check(context.Background())

	assert.Equal(t, Unhealthy, status.Status)
	assert.Equal(t, Healthy, status.Details.Database)
	assert.Equal(t, Unhealthy, status.Details.Redis)
	assert.Equal(t, Unhealthy, status.Details.Kafka) // Kafka check will fail
	assert.WithinDuration(t, time.Now(), status.Details.Timestamp, 1*time.Second)

	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestHealthChecker_Check_KafkaUnhealthy(t *testing.T) {
	t.Parallel()
	_, dbMock, _, redisMock, kafkaWriter, hc := setupMocks(t)

	assert.Equal(t, "localhost:99999", kafkaWriter.Addr.String())

	dbMock.ExpectPing().WillReturnError(nil) // DB Healthy
	redisMock.ExpectPing().SetVal("PONG")    // Redis Healthy

	status := hc.Check(context.Background())

	assert.Equal(t, Unhealthy, status.Status)
	assert.Equal(t, Healthy, status.Details.Database)
	assert.Equal(t, Healthy, status.Details.Redis)
	assert.Equal(t, Unhealthy, status.Details.Kafka)
	assert.WithinDuration(t, time.Now(), status.Details.Timestamp, 1*time.Second)

	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestHealthChecker_Check_MultipleUnhealthy(t *testing.T) {
	t.Parallel()
	_, dbMock, _, redisMock, _, hc := setupMocks(t)

	dbErr := errors.New("db connection failed")
	redisErr := errors.New("redis connection refused")

	dbMock.ExpectPing().WillReturnError(dbErr) // DB Unhealthy
	redisMock.ExpectPing().SetErr(redisErr)    // Redis Unhealthy

	status := hc.Check(context.Background())

	assert.Equal(t, Unhealthy, status.Status)
	assert.Equal(t, Unhealthy, status.Details.Database)
	assert.Equal(t, Unhealthy, status.Details.Redis)
	assert.Equal(t, Unhealthy, status.Details.Kafka)
	assert.WithinDuration(t, time.Now(), status.Details.Timestamp, 1*time.Second)

	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}
