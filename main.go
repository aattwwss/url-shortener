package main

import (
	"context"
	"embed"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"url-shortener/rate"
	"url-shortener/store"
	"url-shortener/zap"

	"github.com/gorilla/mux"
)

var (
	//go:embed templates
	templateFolder embed.FS
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("error loading .env file")
	}
	ctx := context.Background()

	redisUrl := os.Getenv("REDIS_URL")
	redisUsername := os.Getenv("REDIS_USERNAME")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDatabase := os.Getenv("REDIS_DATABASE")
	strategyEnv := os.Getenv("STRATEGY")
	limitEnv := os.Getenv("REQUEST_LIMIT")
	isHTTPS := os.Getenv("IS_HTTPS")

	redisClient, err := store.NewRedisClient(ctx, redisUrl, redisDatabase, redisUsername, redisPassword)
	if err != nil {
		log.Fatalf("error setting up redis: %v", err)
	}

	strategy, err := strconv.Atoi(strategyEnv)
	if err != nil || strategy < 1 || strategy > 4 {
		log.Fatalf("strategy must be from 1 to 4: %v", err)
	}

	limit, err := strconv.Atoi(limitEnv)
	if err != nil || limit < -1 {
		log.Fatalf("limit must be larger than -1: %v", err)
	}

	limiter := rate.NewLimiter(templateFolder, redisClient, strategy, limit)
	z := zap.NewZap(redisClient, isHTTPS, limiter.LimitMiddleware, templateFolder)

	router := mux.NewRouter()
	z.Register(router)

	server := &http.Server{
		Addr:         ":9090",
		WriteTimeout: time.Second * 5,
		ReadTimeout:  time.Second * 5,
		IdleTimeout:  time.Second * 30,
		Handler:      router,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalln("error starting the server: ", err)
	}

	return
}
