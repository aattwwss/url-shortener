package store

import (
	"context"
	"time"
)

type KeyValueStorer interface {
	Set(ctx context.Context, key string, value string, expiry time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}
