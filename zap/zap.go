package zap

import (
	"context"
	"embed"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
	"url-shortener/rate"
	"url-shortener/store"
)

const (
	charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

type UrlPayload struct {
	Original  string
	Shortened string
	Error     string
}

type Zap struct {
	store     store.KeyValueStorage
	isHttps   bool
	limiter   rate.Limiter
	templates embed.FS
}

func NewZap(store store.KeyValueStorage, isHttpsEnv string, limiter rate.Limiter, templates embed.FS) Zap {
	isHttpsEnv = strings.ToLower(isHttpsEnv)
	return Zap{
		store:     store,
		isHttps:   isHttpsEnv == "true" || isHttpsEnv == "yes" || isHttpsEnv == "1",
		limiter:   limiter,
		templates: templates,
	}
}

func (z Zap) Register(router *mux.Router) {
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	router.HandleFunc("/", z.Home).Methods(http.MethodGet)
	router.HandleFunc("/r/{key}", z.Redirect).Methods(http.MethodGet)

	limited := router.PathPrefix("").Subrouter()
	limited.Use(z.limiter.Limit)
	limited.HandleFunc("/", z.Create).Methods(http.MethodPost)
}

func (z Zap) Home(w http.ResponseWriter, _ *http.Request) {
	homeTemplate, err := template.ParseFS(z.templates, "templates/index.html")
	if err != nil {
		return
	}
	_ = homeTemplate.Execute(w, UrlPayload{})
}

func (z Zap) Create(w http.ResponseWriter, r *http.Request) {
	homeTemplate, err := template.ParseFS(z.templates, "templates/index.html")
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
	scheme := z.getHttpScheme()
	shortened := fmt.Sprintf("%s://%s/r/%s", scheme, r.Host, key)
	payload := UrlPayload{input, shortened, ""}

	err = z.store.Set(context.Background(), key, longUrl, toDuration(duration))
	if err != nil {
		log.Println("Error setting longUrl")
		payload := UrlPayload{input, "", "Something went wrong."}
		_ = homeTemplate.Execute(w, payload)
		return
	}

	_ = homeTemplate.Execute(w, payload)
}

func (z Zap) Redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	longUrl, err := z.store.Get(context.Background(), key)
	if err != nil {
		log.Printf("Error retrieving url from %v: %v\n", key, err)
		return
	}
	http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
}

func (z Zap) getHttpScheme() string {
	if z.isHttps {
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
