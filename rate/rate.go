package rate

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Strategy int

const (
	FixedWindow = Strategy(iota + 1)
	SlidingWindow
	TokenBucket
	LeakyBucket
)

type contextKey int

const (
	contextKeyIdentifier = contextKey(iota)
)

func SetIdentifier(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, contextKeyIdentifier, value)
}

const (
	keyPrefix = "rate"
)

type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Incr(ctx context.Context, key string) (int64, error)
	IncrWithExpiry(ctx context.Context, key string, expiry time.Duration) (int64, error)
}

type Limiter struct {
	templates embed.FS
	store     Store
	strategy  Strategy
	limit     int // number of requests per minute
}

func NewLimiter(templates embed.FS, store Store, strategy int, limit int) Limiter {
	return Limiter{templates: templates, store: store, strategy: Strategy(strategy), limit: limit}
}

func (l Limiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.Allow(r.Context(), r) {
			next.ServeHTTP(w, r)
		} else {
			l.ServeError(w, r)
		}
	})
}

func (l Limiter) Allow(ctx context.Context, r *http.Request) bool {
	if l.limit == -1 {
		return true
	}

	identifier := r.Context().Value(contextKeyIdentifier).(string)
	if l.strategy == FixedWindow {
		return l.FixedWindow(ctx, identifier)
	} else if l.strategy == SlidingWindow {
		return l.SlidingWindow(ctx, identifier)
	} else if l.strategy == TokenBucket {
		return l.TokenBucket(ctx, identifier)
	} else if l.strategy == LeakyBucket {
		return l.LeakyBucket(ctx, identifier)
	} else {
		return l.FixedWindow(ctx, identifier)
	}
}

func (l Limiter) ServeError(w http.ResponseWriter, _ *http.Request) {
	homeTemplate, err := template.ParseFS(l.templates, "templates/index.html")
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusTooManyRequests)
	payload := map[string]string{"Error": "Please try again later."}
	_ = homeTemplate.Execute(w, payload)
	return
}

func (l Limiter) FixedWindow(ctx context.Context, identifier string) bool {
	key := fmt.Sprintf("%s:%s:%v", keyPrefix, identifier, time.Now().Minute())
	count, _ := l.store.Get(ctx, key)
	countInt, err := strconv.Atoi(count)
	if err != nil || countInt == 0 {
		expiry := time.Duration(60-time.Now().Second()) * time.Second
		_, err = l.store.IncrWithExpiry(ctx, key, expiry)
		if err != nil {
			log.Printf("Error incrementing key with expiry %v: %v\n", key, err)
			return false
		}
		return true
	}

	if countInt < l.limit {
		_, err = l.store.Incr(ctx, key)
		if err != nil {
			log.Printf("Error incrementing key %v: %v\n", key, err)
			return false
		}
		return true
	}
	return false
}

func (l Limiter) SlidingWindow(ctx context.Context, identifier string) bool {
	return true
}
func (l Limiter) TokenBucket(ctx context.Context, identifier string) bool {
	return true
}
func (l Limiter) LeakyBucket(ctx context.Context, identifier string) bool {
	return true
}
