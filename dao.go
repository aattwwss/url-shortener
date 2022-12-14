package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisDAO struct {
	client *redis.Client
}

func NewRedisDAO(ctx context.Context, redisURL string, db string, username string, password string) (*RedisDAO, error) {
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

	return &RedisDAO{
		client: client,
	}, nil
}

func (dao *RedisDAO) Set(ctx context.Context, key string, value string, expiry time.Duration) error {
	// Set the value for the given key in Redis.
	err := dao.client.Set(ctx, key, value, expiry).Err()
	if err != nil {
		return fmt.Errorf("failed to set value in Redis: %w", err)
	}
	return nil
}

func (dao *RedisDAO) Get(ctx context.Context, key string) (string, error) {
	// Get the value for the given key from Redis.
	val, err := dao.client.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("failed to get value from Redis: %w", err)
	}
	return val, nil
}
