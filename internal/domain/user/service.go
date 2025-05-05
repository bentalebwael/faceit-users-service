package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service implements the core business logic for user management
type Service struct {
	repo      Repository
	publisher Publisher
	logger    *slog.Logger
}

func NewService(repo Repository, publisher Publisher, logger *slog.Logger) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *Service) CreateUser(ctx context.Context, user *User) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user.ID = uuid.New()
	user.Password = string(hashedPassword)
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	if _, err := s.repo.GetByEmail(ctx, user.Email); err == nil {
		return nil, ErrEmailTaken
	}

	if _, err := s.repo.GetByNickname(ctx, user.Nickname); err == nil {
		return nil, ErrNicknameTaken
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	if err := s.publisher.PublishCreatedUser(ctx, user); err != nil {
		s.logger.Warn("failed to publish user created event",
			"error", err,
			"user_id", user.ID,
		)
	}

	return user, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, updatedUser *User) (*User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) { // Assuming repo returns ErrNotFound
			return nil, err
		}
		return nil, fmt.Errorf("failed to get user for update: %w", err)
	}

	if updatedUser.FirstName != "" && user.FirstName != updatedUser.FirstName {
		user.FirstName = updatedUser.FirstName
	}
	if updatedUser.LastName != "" && user.LastName != updatedUser.LastName {
		user.LastName = updatedUser.LastName
	}
	if updatedUser.Nickname != "" && user.Nickname != updatedUser.Nickname {
		user.Nickname = updatedUser.Nickname
	}
	if updatedUser.Country != "" && user.Country != updatedUser.Country {
		user.Country = updatedUser.Country
	}
	if updatedUser.Email != "" && user.Email != updatedUser.Email {
		if existingUser, err := s.repo.GetByEmail(ctx, updatedUser.Email); err == nil && existingUser.ID != id {
			return nil, ErrEmailTaken
		}
		user.Email = updatedUser.Email
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user changes: %w", err)
	}

	if err := s.publisher.PublishUpdatedUser(ctx, user); err != nil {
		s.logger.Warn("failed to publish user updated event",
			"error", err,
			"user_id", user.ID,
		)
	}

	return user, nil
}

func (s *Service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return err // Return ErrNotFound directly if user doesn't exist
		}
		return fmt.Errorf("failed to get user for deletion: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user from repository: %w", err)
	}

	if err := s.publisher.PublishDeletedUser(ctx, user); err != nil {
		s.logger.Warn("failed to publish user deleted event",
			"error", err,
			"user_id", user.ID,
		)
	}

	return nil
}

func (s *Service) ListUsers(ctx context.Context, params ListParams) ([]User, bool, int64, error) {
	allowedFilters := map[string]struct{}{
		"first_name": {},
		"last_name":  {},
		"nickname":   {},
		"email":      {},
		"country":    {},
	}

	validFilters := make([]Filter, 0, len(params.Filters))
	for _, filter := range params.Filters {
		if _, ok := allowedFilters[filter.Field]; ok {
			validFilters = append(validFilters, filter)
		}
	}
	params.Filters = validFilters
	if params.OrderBy != "" {
		if _, ok := allowedFilters[params.OrderBy]; !ok && params.OrderBy != "created_at" && params.OrderBy != "updated_at" {
			params.OrderBy = "created_at" // Default to created_at if invalid
		}
	}

	users, totalCount, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, false, 0, fmt.Errorf("failed to list users: %w", err)
	}

	hasMore := false
	if params.Limit > 0 && totalCount > int64(params.Offset+params.Limit) {
		hasMore = true
	}

	return users, hasMore, totalCount, nil
}
