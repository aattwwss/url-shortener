package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"math/rand"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

var (
	redisDao *RedisDAO

	homeTemplate = template.Must(template.ParseFiles("template/index.html"))
	//resultTemplate = template.Must(template.ParseFiles("template/result.html"))
)

const (
	charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

type UrlPayload struct {
	Original  string
	Shortened string
}

func getHTTPSchemeFromRequest(r *http.Request) string {
	if r.TLS == nil {
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
	err := homeTemplate.Execute(w, UrlPayload{})
	if err != nil {
		return
	}
}

func create(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	key := randomString(7)
	scheme := getHTTPSchemeFromRequest(r)
	shortened := fmt.Sprintf("%s://%s/%s", scheme, r.Host, key)
	payload := UrlPayload{url, shortened}

	err := redisDao.Set(context.Background(), key, url, 1*time.Hour)
	if err != nil {
		log.Println("Error setting url")
		return
	}

	err = homeTemplate.Execute(w, payload)
	if err != nil {
		return
	}
}

func redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	url, err := redisDao.Get(context.Background(), key)
	if err != nil {
		log.Printf("Error retrieving url from %v: %v\n", key, err)
		return
	}
	http.Redirect(w, r, url, http.StatusMovedPermanently)
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
