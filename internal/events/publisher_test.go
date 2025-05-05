package events

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
)

// mockKafkaWriter simulates a Kafka writer for testing
type mockKafkaWriter struct {
	messages []kafka.Message
}

func (m *mockKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	m.messages = append(m.messages, msgs...)
	return nil
}

func (m *mockKafkaWriter) Close() error {
	return nil
}

func newMockKafkaWriter() *mockKafkaWriter {
	return &mockKafkaWriter{
		messages: make([]kafka.Message, 0),
	}
}

func TestUserEventPublisher_createUserEvent(t *testing.T) {
	mockWriter := newMockKafkaWriter()
	publisher := NewUserEventPublisher(mockWriter)

	testUser := &user.User{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	tests := []struct {
		name      string
		user      *user.User
		eventType EventType
	}{
		{
			name:      "create event",
			user:      testUser,
			eventType: EventTypeCreated,
		},
		{
			name:      "update event",
			user:      testUser,
			eventType: EventTypeUpdated,
		},
		{
			name:      "delete event",
			user:      testUser,
			eventType: EventTypeDeleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := publisher.createUserEvent(tt.user, tt.eventType)

			if event.Type != tt.eventType {
				t.Errorf("createUserEvent() event type = %v, want %v", event.Type, tt.eventType)
			}
			if event.User != tt.user {
				t.Errorf("createUserEvent() user = %v, want %v", event.User, tt.user)
			}
			if event.ID == "" {
				t.Error("createUserEvent() event ID is empty")
			}
			if event.Timestamp.IsZero() {
				t.Error("createUserEvent() timestamp is zero")
			}
			if event.Version != "1.0" {
				t.Errorf("createUserEvent() version = %v, want 1.0", event.Version)
			}
		})
	}
}

func TestUserEventPublisher_Publish(t *testing.T) {
	mockWriter := newMockKafkaWriter()
	publisher := NewUserEventPublisher(mockWriter)

	testUser := &user.User{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	tests := []struct {
		name      string
		event     *Event
		wantError bool
	}{
		{
			name: "valid event",
			event: &Event{
				Type:      EventTypeCreated,
				ID:        uuid.New().String(),
				User:      testUser,
				Timestamp: time.Now().UTC(),
				Version:   "1.0",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := publisher.Publish(context.Background(), tt.event)
			if (err != nil) != tt.wantError {
				t.Errorf("Publish() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				// Verify the last message
				lastMsg := mockWriter.messages[len(mockWriter.messages)-1]

				// Check message key
				if string(lastMsg.Key) != tt.event.ID {
					t.Errorf("Message key = %s, want %s", string(lastMsg.Key), tt.event.ID)
				}

				// Check headers
				hasEventID := false
				hasEventType := false
				hasVersion := false
				for _, h := range lastMsg.Headers {
					switch h.Key {
					case "event-id":
						hasEventID = true
						if string(h.Value) != tt.event.ID {
							t.Errorf("Header event-id = %s, want %s", string(h.Value), tt.event.ID)
						}
					case "event-type":
						hasEventType = true
						if string(h.Value) != string(tt.event.Type) {
							t.Errorf("Header event-type = %s, want %s", string(h.Value), tt.event.Type)
						}
					case "event-schema-version":
						hasVersion = true
						if string(h.Value) != tt.event.Version {
							t.Errorf("Header event-schema-version = %s, want %s", string(h.Value), tt.event.Version)
						}
					}
				}

				if !hasEventID {
					t.Error("Message headers missing event-id")
				}
				if !hasEventType {
					t.Error("Message headers missing event-type")
				}
				if !hasVersion {
					t.Error("Message headers missing event-schema-version")
				}

				// Verify payload
				var decodedEvent Event
				if err := json.Unmarshal(lastMsg.Value, &decodedEvent); err != nil {
					t.Errorf("Failed to decode message payload: %v", err)
					return
				}

				if decodedEvent.Type != tt.event.Type {
					t.Errorf("Decoded event type = %v, want %v", decodedEvent.Type, tt.event.Type)
				}
				if decodedEvent.ID != tt.event.ID {
					t.Errorf("Decoded event ID = %v, want %v", decodedEvent.ID, tt.event.ID)
				}
				if decodedEvent.User.ID != tt.event.User.ID {
					t.Errorf("Decoded user ID = %v, want %v", decodedEvent.User.ID, tt.event.User.ID)
				}
			}
		})
	}
}

func TestUserEventPublisher_PublishMethods(t *testing.T) {
	mockWriter := newMockKafkaWriter()
	publisher := NewUserEventPublisher(mockWriter)

	testUser := &user.User{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	tests := []struct {
		name     string
		method   func(context.Context, *user.User) error
		wantType EventType
	}{
		{
			name:     "publish created user",
			method:   publisher.PublishCreatedUser,
			wantType: EventTypeCreated,
		},
		{
			name:     "publish updated user",
			method:   publisher.PublishUpdatedUser,
			wantType: EventTypeUpdated,
		},
		{
			name:     "publish deleted user",
			method:   publisher.PublishDeletedUser,
			wantType: EventTypeDeleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgCount := len(mockWriter.messages)

			err := tt.method(context.Background(), testUser)
			if err != nil {
				t.Errorf("Publishing method failed: %v", err)
				return
			}

			if len(mockWriter.messages) != msgCount+1 {
				t.Error("No message was published")
				return
			}

			// Decode the last message
			lastMsg := mockWriter.messages[len(mockWriter.messages)-1]
			var event Event
			if err := json.Unmarshal(lastMsg.Value, &event); err != nil {
				t.Errorf("Failed to decode message payload: %v", err)
				return
			}

			if event.Type != tt.wantType {
				t.Errorf("Event type = %v, want %v", event.Type, tt.wantType)
			}
			if event.User.ID != testUser.ID {
				t.Errorf("User ID = %v, want %v", event.User.ID, testUser.ID)
			}
		})
	}
}
