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

func getHTTPSchemeFromRequest() string {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "dev" {
		return "http"
	}
	return "https"
}

func randomString(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)

	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}

	return string(b)
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
	longUrl := input
	if input != "" && !strings.HasPrefix(input, "https://") && !strings.HasPrefix(input, "http://") {
		longUrl = "https://" + longUrl
	}

	_, err = url.ParseRequestURI(longUrl)
	if err != nil {
		payload := UrlPayload{input, "", "Invalid url."}
		_ = homeTemplate.Execute(w, payload)
		return
	}

	key := randomString(7)
	scheme := getHTTPSchemeFromRequest()
	shortened := fmt.Sprintf("%s://%s/%s", scheme, r.Host, key)
	payload := UrlPayload{longUrl, shortened, ""}

	err = redisDao.Set(context.Background(), key, longUrl, 1*time.Hour)
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

	router.HandleFunc("/", home).Methods("GET")
	router.HandleFunc("/", create).Methods("POST")
	router.HandleFunc("/{key}", redirect).Methods("GET")

	err = http.ListenAndServe(":9090", router)
	if err != nil {
		log.Fatalln("There's an error with the server", err)
	}
}
