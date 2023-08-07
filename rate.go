package main

import (
	"fmt"
	"net/http"
	"url-shortener/store"
)

type Limiter struct {
	redisDao *store.RedisClient
	strategy Strategy
	limit    int // number of requests per minute
}

type Strategy func(limit int, r *http.Request) bool

func (l Limiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.Allow() {
			fmt.Println("middleware1")
			next.ServeHTTP(w, r)
		}
	})
}

func (l Limiter) Allow() bool {
	return true
}
