package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

var (
	//go:embed templates
	templateFolder embed.FS
	redisDao       *RedisDAO
)

const (
	charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

type UrlPayload struct {
	Original  string
	Shortened string
	Error     string
}

func getHttpScheme() string {
	isHttps := os.Getenv("IS_HTTPS")
	if isHttps == "true" {
		return "https"
	}
	return "http"
}

func randomString(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)

	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}

	return string(b)
}

func toDuration(durationUnit string) time.Duration {
	switch strings.ToLower(durationUnit) {
	case "minute":
		return time.Minute
	case "hour":
		return time.Hour
	case "day":
		return time.Hour * 24
	case "week":
		return time.Hour * 24 * 7
	case "month":
		return time.Hour * 24 * 30
	case "year":
		return time.Hour * 24 * 365
	default:
		return time.Hour
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate, err := template.ParseFS(templateFolder, "templates/index.html")
	if err != nil {
		return
	}
	_ = homeTemplate.Execute(w, UrlPayload{})
}

func create(w http.ResponseWriter, r *http.Request) {
	homeTemplate, err := template.ParseFS(templateFolder, "templates/index.html")
	if err != nil {
		return
	}
	input := r.FormValue("url")
	duration := r.FormValue("duration")
	longUrl := input
	if longUrl != "" && !strings.HasPrefix(longUrl, "https://") && !strings.HasPrefix(longUrl, "http://") {
		longUrl = "https://" + longUrl
	}

	_, err = url.ParseRequestURI(longUrl)
	if err != nil {
		payload := UrlPayload{input, "", "Invalid url."}
		_ = homeTemplate.Execute(w, payload)
		return
	}

	key := randomString(7)
	scheme := getHttpScheme()
	shortened := fmt.Sprintf("%s://%s/r/%s", scheme, r.Host, key)
	payload := UrlPayload{input, shortened, ""}

	err = redisDao.Set(context.Background(), key, longUrl, toDuration(duration))
	if err != nil {
		log.Println("Error setting longUrl")
		return
	}

	_ = homeTemplate.Execute(w, payload)
}

func redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	longUrl, err := redisDao.Get(context.Background(), key)
	if err != nil {
		log.Printf("Error retrieving url from %v: %v\n", key, err)
		return
	}
	http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
}

func rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("middleware1")
		next.ServeHTTP(w, r)
	})
}

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
	redisDao, err = NewRedisDAO(ctx, redisUrl, redisDatabase, redisUsername, redisPassword)
	if err != nil {
		log.Fatalf("Error setting up redis: %v", err)
	}

	router := mux.NewRouter()
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	router.HandleFunc("/", home).Methods(http.MethodGet)
	router.HandleFunc("/r/{key}", redirect).Methods(http.MethodGet)

	limited := router.PathPrefix("").Subrouter()
	limited.Use(rateLimit)
	limited.HandleFunc("/", create).Methods(http.MethodPost)

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
