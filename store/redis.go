package store

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(ctx context.Context, redisURL string, db string, username string, password string) (*RedisClient, error) {
	// Create a new Redis client.
	dbInt, err := strconv.Atoi(db)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Username: username,
		Password: password,
		DB:       dbInt,
	})

	// Test the connection to the Redis server.
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: client,
	}, nil
}

// Set the value for the given key in Redis.
func (r *RedisClient) Set(ctx context.Context, key string, value string, expiry time.Duration) error {
	err := r.client.Set(ctx, key, value, expiry).Err()
	if err != nil {
		return fmt.Errorf("failed to set value in Redis: %w", err)
	}
	return nil
}

// Get the value for the given key from Redis.
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("failed to get value from Redis: %w", err)
	}
	return val, nil
}

// Incr increments the value for the given key in Redis.
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// IncrWithExpiry increments the value for the given key in Redis and sets the expiry.
func (r *RedisClient) IncrWithExpiry(ctx context.Context, key string, expiry time.Duration) (int64, error) {
	pipe := r.client.Pipeline()
	intCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiry)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment with expiry value in Redis: %w", err)
	}
	return intCmd.Val(), nil
}
