package store

import (
	"context"
	"time"
)

type KeyValueStorage interface {
	Set(ctx context.Context, key string, value string, expiry time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}
