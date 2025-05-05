package redis

import (
	"context"
	"fmt"

	"github.com/bentalebwael/faceit-users-service/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewClient creates and configures a new Redis client
func NewClient(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Redis.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("error connecting to redis: %w", err)
	}

	return client, nil
}

func Close(client *redis.Client) error {
	if client != nil {
		if err := client.Close(); err != nil {
			return fmt.Errorf("error closing redis connection: %w", err)
		}
	}
	return nil
}
