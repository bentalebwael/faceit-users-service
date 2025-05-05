package user

import (
	"context"
)

// Publisher defines the interface for publishing user events
type Publisher interface {
	PublishCreatedUser(ctx context.Context, User *User) error
	PublishUpdatedUser(ctx context.Context, User *User) error
	PublishDeletedUser(ctx context.Context, User *User) error
}
