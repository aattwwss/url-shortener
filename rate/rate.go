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
	"url-shortener/store"
)

type Limiter struct {
	templates   embed.FS
	redisClient *store.RedisClient
	strategy    Strategy
	limit       int // number of requests per minute
}

func NewLimiter(templates embed.FS, redisClient *store.RedisClient, strategy int, limit int) Limiter {
	return Limiter{templates: templates, redisClient: redisClient, strategy: toStrategy(strategy), limit: limit}
}

type Strategy int

const (
	_ Strategy = iota
	FixedWindow
	SlidingWindow
	TokenBucket
	LeakyBucket
)

func toStrategy(i int) Strategy {
	switch i {
	case 1:
		return FixedWindow
	case 2:
		return SlidingWindow
	case 3:
		return TokenBucket
	case 4:
		return LeakyBucket
	default:
		return FixedWindow
	}
}

const (
	keyPrefix = "rate"
)

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
	identifier := r.Header.Get("Cf-Connecting-Ip")
	if l.strategy == FixedWindow {
		return l.FixedWindow(ctx, identifier)
	} else if l.strategy == SlidingWindow {
		return l.SlidingWindow(ctx, identifier)
	} else if l.strategy == TokenBucket {
		return l.TokenBucket(ctx, identifier)
	} else if l.strategy == LeakyBucket {
		return l.LeakyBucket(ctx, identifier)
	}
	return false
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
	count, _ := l.redisClient.Get(ctx, key)
	countInt, err := strconv.Atoi(count)
	if err != nil || countInt == 0 {
		expiry := time.Duration(60-time.Now().Second()) * time.Second
		_, err = l.redisClient.IncrWithExpiry(ctx, key, expiry)
		if err != nil {
			log.Printf("Error incrementing key with expiry %v: %v\n", key, err)
			return false
		}
		return true
	}

	if countInt < l.limit {
		_, err = l.redisClient.Incr(ctx, key)
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
