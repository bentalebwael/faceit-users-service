package kafka

import (
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/bentalebwael/faceit-users-service/internal/config"
)

// NewProducer creates a new Kafka writer (producer).
func NewProducer(cfg *config.Config, log *slog.Logger) (*kafka.Writer, error) {
	conn, err := kafka.Dial("tcp", cfg.Kafka.Brokers)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	err = createTopicIfNotExists(conn, cfg, log)
	if err != nil {
		return nil, err
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers),
		Topic:        cfg.Kafka.EventTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        true,
		WriteTimeout: cfg.Kafka.WriteTimeout,
	}

	return writer, nil
}

func Close(writer *kafka.Writer) error {
	if writer != nil {
		return writer.Close()
	}
	return nil
}

func createTopicIfNotExists(conn *kafka.Conn, cfg *config.Config, log *slog.Logger) error {
	var partitions []kafka.Partition
	var err error

	for i := range 20 {
		log.Info("Trying to read Kafka partitions", "attempt", i+1)
		partitions, err = conn.ReadPartitions(cfg.Kafka.EventTopic)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	if err != nil {
		return err
	}

	if len(partitions) == 0 {
		err = conn.CreateTopics(kafka.TopicConfig{
			Topic:             cfg.Kafka.EventTopic,
			NumPartitions:     cfg.Kafka.NumPartitions,
			ReplicationFactor: cfg.Kafka.ReplicationFactor,
		})
		if err != nil {
			return err
		}
		log.Info("Created Kafka topic", "topic", cfg.Kafka.EventTopic)
	}
	return nil
}
