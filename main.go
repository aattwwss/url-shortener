package main

import (
	"context"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"time"
	"url-shortener/store"
	"url-shortener/zap"

	"github.com/gorilla/mux"
)

func main() {
	rateLimit := Limiter{}

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

	z := zap.NewZap(redisClient, os.Getenv("IS_HTTPS"))

	router := mux.NewRouter()
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	router.HandleFunc("/", z.Home).Methods(http.MethodGet)
	router.HandleFunc("/r/{key}", z.Redirect).Methods(http.MethodGet)

	limited := router.PathPrefix("").Subrouter()
	limited.Use(rateLimit.Limit)
	limited.HandleFunc("/", z.Create).Methods(http.MethodPost)

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
