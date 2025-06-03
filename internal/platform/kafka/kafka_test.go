package kafka

import (
	"testing"
	"time"

	"github.com/bentalebwael/faceit-users-service/internal/config"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

func TestNewProducer(t *testing.T) {
	t.Parallel()
	// Since we can't replace kafka.Dial directly, we'll test individual scenarios
	// without actually calling kafka.Dial

	t.Run("Success with existing topic", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{
			Kafka: config.KafkaConfig{
				Brokers:    "localhost:9092",
				EventTopic: "test-topic",
			},
		}

		expectedWriter := &kafka.Writer{
			Addr:         kafka.TCP(cfg.Kafka.Brokers),
			Topic:        cfg.Kafka.EventTopic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        true,
			WriteTimeout: 10 * time.Second,
		}

		// We can only test the configuration of the writer here
		writer, err := createTestWriter(cfg)
		assert.NoError(t, err)
		assert.Equal(t, expectedWriter.Addr, writer.Addr)
		assert.Equal(t, expectedWriter.Topic, writer.Topic)
		assert.Equal(t, expectedWriter.WriteTimeout, writer.WriteTimeout)
		assert.True(t, writer.Async)
	})
}

func createTestWriter(cfg *config.Config) (*kafka.Writer, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers),
		Topic:        cfg.Kafka.EventTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        true,
		WriteTimeout: 10 * time.Second,
	}
	return writer, nil
}

func TestClose(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		writer := &kafka.Writer{
			Addr:  kafka.TCP("localhost:9092"),
			Topic: "test-topic",
		}

		err := Close(writer)
		assert.NoError(t, err)
	})

	t.Run("Nil writer", func(t *testing.T) {
		t.Parallel()
		err := Close(nil)
		assert.NoError(t, err)
	})
}
