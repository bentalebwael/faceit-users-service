package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
)

type EventType string

const (
	EventTypeCreated EventType = "created"
	EventTypeUpdated EventType = "updated"
	EventTypeDeleted EventType = "deleted"
)

type Event struct {
	Type      EventType  `json:"type"`
	ID        string     `json:"id"`        // Event ID
	User      *user.User `json:"User"`      // Event payload
	Timestamp time.Time  `json:"timestamp"` // When the event occurred
	Version   string     `json:"version"`   // Event schema version
}

// KafkaWriter interface defines the methods we need from kafka.Writer
type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type UserEventPublisher struct {
	writer KafkaWriter
}

// NewUserEventPublisher creates a new Kafka event publisher for user events.
func NewUserEventPublisher(writer KafkaWriter) *UserEventPublisher {
	return &UserEventPublisher{
		writer: writer,
	}
}

func (p *UserEventPublisher) PublishCreatedUser(ctx context.Context, User *user.User) error {
	event := p.createUserEvent(User, EventTypeCreated)
	return p.Publish(ctx, event)
}

func (p *UserEventPublisher) PublishUpdatedUser(ctx context.Context, User *user.User) error {
	event := p.createUserEvent(User, EventTypeUpdated)
	return p.Publish(ctx, event)
}

func (p *UserEventPublisher) PublishDeletedUser(ctx context.Context, User *user.User) error {
	event := p.createUserEvent(User, EventTypeDeleted)
	return p.Publish(ctx, event)
}

// Publish sends a user event to the message broker, implementing the user.Publisher interface.
func (p *UserEventPublisher) Publish(ctx context.Context, event *Event) error {
	payload, err := json.Marshal(*event)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.ID),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event-id", Value: []byte(event.ID)},
			{Key: "event-type", Value: []byte(event.Type)},
			{Key: "event-schema-version", Value: []byte(event.Version)},
		},
	}
	return p.writer.WriteMessages(ctx, msg)
}

func (p *UserEventPublisher) createUserEvent(User *user.User, eventType EventType) *Event {
	return &Event{
		Type:      eventType,
		ID:        uuid.New().String(),
		User:      User,
		Timestamp: time.Now().UTC(),
		Version:   "1.0", // Event schema version
	}
}
