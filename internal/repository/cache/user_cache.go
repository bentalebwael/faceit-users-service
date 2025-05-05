package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/bentalebwael/faceit-users-service/internal/config"
	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
)

const (
	// Cache key prefixes
	userKeyPrefix  = "user:"
	emailKeyPrefix = "user:email:"
	nickKeyPrefix  = "user:nick:"
)

// CacheDecorator wraps a user.Repository with caching functionality
type CacheDecorator struct {
	repo  user.Repository
	redis *redis.Client
	ttl   time.Duration
}

func NewCacheDecorator(repo user.Repository, redis *redis.Client, cfg *config.RedisConfig) *CacheDecorator {
	return &CacheDecorator{
		repo:  repo,
		redis: redis,
		ttl:   cfg.CacheTTL,
	}
}

func (c *CacheDecorator) Create(ctx context.Context, u *user.User) error {
	if err := c.repo.Create(ctx, u); err != nil {
		return err
	}

	if err := c.cacheUser(ctx, u); err != nil {
		return fmt.Errorf("error caching user: %w", err)
	}

	return nil
}

func (c *CacheDecorator) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if u, err := c.getUserFromCache(ctx, userKey(id)); err == nil {
		return u, nil
	}

	u, err := c.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := c.cacheUser(ctx, u); err != nil {
		return nil, fmt.Errorf("error caching user: %w", err)
	}

	return u, nil
}

func (c *CacheDecorator) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if u, err := c.getUserFromCache(ctx, emailKey(email)); err == nil {
		return u, nil
	}

	u, err := c.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if err := c.cacheUser(ctx, u); err != nil {
		return nil, fmt.Errorf("error caching user: %w", err)
	}

	return u, nil
}

func (c *CacheDecorator) GetByNickname(ctx context.Context, nickname string) (*user.User, error) {
	if u, err := c.getUserFromCache(ctx, nickKey(nickname)); err == nil {
		return u, nil
	}

	u, err := c.repo.GetByNickname(ctx, nickname)
	if err != nil {
		return nil, err
	}

	if err := c.cacheUser(ctx, u); err != nil {
		return nil, fmt.Errorf("error caching user: %w", err)
	}

	return u, nil
}

func (c *CacheDecorator) Update(ctx context.Context, u *user.User) error {
	oldUser, err := c.repo.GetByID(ctx, u.ID)
	if err != nil {
		return err
	}

	if err := c.repo.Update(ctx, u); err != nil {
		return err
	}

	if err := c.invalidateUserCache(ctx, oldUser); err != nil {
		return fmt.Errorf("error invalidating cache: %w", err)
	}

	updatedUser, err := c.repo.GetByID(ctx, u.ID)
	if err != nil {
		return err
	}
	if err := c.cacheUser(ctx, updatedUser); err != nil {
		return fmt.Errorf("error caching updated user: %w", err)
	}

	return nil
}

func (c *CacheDecorator) Delete(ctx context.Context, id uuid.UUID) error {
	u, err := c.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := c.repo.Delete(ctx, id); err != nil {
		return err
	}

	if err := c.invalidateUserCache(ctx, u); err != nil {
		return fmt.Errorf("error invalidating cache: %w", err)
	}

	return nil
}

func (c *CacheDecorator) List(ctx context.Context, params user.ListParams) ([]user.User, int64, error) {
	// Currently bypasses cache for list operations.
	// Caching list results requires more complex invalidation strategies.
	return c.repo.List(ctx, params)
}

func (c *CacheDecorator) cacheUser(ctx context.Context, u *user.User) error {
	data, err := json.Marshal(u)
	if err != nil {
		return fmt.Errorf("error marshaling user: %w", err)
	}

	pipe := c.redis.Pipeline()
	pipe.Set(ctx, userKey(u.ID), data, c.ttl)
	pipe.Set(ctx, emailKey(u.Email), data, c.ttl)
	pipe.Set(ctx, nickKey(u.Nickname), data, c.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("error executing cache pipeline: %w", err)
	}

	return nil
}

func (c *CacheDecorator) getUserFromCache(ctx context.Context, key string) (*user.User, error) {
	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var u user.User
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, fmt.Errorf("error unmarshaling user: %w", err)
	}

	return &u, nil
}

func (c *CacheDecorator) invalidateUserCache(ctx context.Context, u *user.User) error {
	pipe := c.redis.Pipeline()
	pipe.Del(ctx, userKey(u.ID))
	pipe.Del(ctx, emailKey(u.Email))
	pipe.Del(ctx, nickKey(u.Nickname))

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("error executing cache invalidation: %w", err)
	}

	return nil
}

func userKey(id uuid.UUID) string {
	return fmt.Sprintf("%s%s", userKeyPrefix, id.String())
}

func emailKey(email string) string {
	return fmt.Sprintf("%s%s", emailKeyPrefix, email)
}

func nickKey(nickname string) string {
	return fmt.Sprintf("%s%s", nickKeyPrefix, nickname)
}
