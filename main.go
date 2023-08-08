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
		log.Fatalln("Error loading .env file")
	}

	redisUrl := os.Getenv("REDIS_URL")
	redisUsername := os.Getenv("REDIS_USERNAME")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDatabase := os.Getenv("REDIS_DATABASE")

	ctx := context.Background()
	redisClient, err := store.NewRedisClient(ctx, redisUrl, redisDatabase, redisUsername, redisPassword)
	if err != nil {
		log.Fatalf("Error setting up redis: %v", err)
	}

	strategyEnv := os.Getenv("STRATEGY")
	limitEnv := os.Getenv("REQUEST_LIMIT")
	strategy, err := strconv.Atoi(strategyEnv)
	if err != nil {
		log.Fatalf("strategy must be from 1 to 4: %v", err)
	}
	limit, err := strconv.Atoi(limitEnv)
	if err != nil {
		log.Fatalf("limit must be a number: %v", err)
	}
	limiter := rate.NewLimiter(templateFolder, redisClient, strategy, limit)
	z := zap.NewZap(redisClient, os.Getenv("IS_HTTPS"), limiter, templateFolder)

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
		log.Fatalln("There's an error with the server", err)
	}

	return
}
