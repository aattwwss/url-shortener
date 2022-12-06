package main

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

type RedisDAO struct {
	client *redis.Client
}

func NewRedisDAO(redisURL string, db int, password string) (*RedisDAO, error) {
	// Create a new Redis client.
	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: password,
		DB:       db,
	})

	// Test the connection to the Redis server.
	_, err := client.Ping().Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisDAO{
		client: client,
	}, nil
}

func (dao *RedisDAO) Set(key string, value string, expiry time.Duration) error {
	// Set the value for the given key in Redis.
	err := dao.client.Set(key, value, expiry).Err()
	if err != nil {
		return fmt.Errorf("failed to set value in Redis: %w", err)
	}
	return nil
}

func (dao *RedisDAO) Get(key string) (string, error) {
	// Get the value for the given key from Redis.
	val, err := dao.client.Get(key).Result()
	if err != nil {
		return "", fmt.Errorf("failed to get value from Redis: %w", err)
	}
	return val, nil
}
