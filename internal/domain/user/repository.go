package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for user persistence operations
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByNickname(ctx context.Context, nickname string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams) ([]User, int64, error)
}

// Filter represents a single filter condition
type Filter struct {
	Field string
	Value string
}

// ListParams defines the parameters for listing users using limit/offset
type ListParams struct {
	Limit     int
	Offset    int
	Filters   []Filter
	OrderBy   string // Field to order by (e.g., "created_at", "email")
	OrderDesc bool
}
