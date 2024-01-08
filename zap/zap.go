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
)

const (
	charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

type urlStore interface {
	Set(ctx context.Context, key string, value string, expiry time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}

type urlPayload struct {
	Shortened string
	Error     string
}

type Zap struct {
	limitMiddleware func(next http.Handler) http.Handler
	store           urlStore
	isHttps         bool
	templates       embed.FS
}

func NewZap(store urlStore, isHttpsEnv string, templates embed.FS) Zap {
	isHttpsEnv = strings.ToLower(isHttpsEnv)
	return Zap{
		store:     store,
		isHttps:   isHttpsEnv == "true" || isHttpsEnv == "yes" || isHttpsEnv == "1",
		templates: templates,
	}
}

func (z Zap) NewRouter() *mux.Router {
	router := mux.NewRouter()
	z.register(router)
	return router
}

func (z Zap) Home(w http.ResponseWriter, _ *http.Request) {
	homeTemplate, err := template.ParseFS(z.templates, "templates/index.html")
	if err != nil {
		return
	}
	_ = homeTemplate.Execute(w, "")
}

func (z Zap) Create(w http.ResponseWriter, r *http.Request) {
	resultTemplate, err := template.ParseFS(z.templates, "templates/result.html")
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
		payload := urlPayload{"", "Invalid url."}
		_ = resultTemplate.Execute(w, payload)
		return
	}

	key := randomString(7)
	scheme := z.getHttpScheme()
	shortened := fmt.Sprintf("%s://%s/r/%s", scheme, r.Host, key)
	payload := urlPayload{shortened, ""}

	err = z.store.Set(context.Background(), key, longUrl, toDuration(duration))
	if err != nil {
		log.Println("Error setting longUrl")
		payload := urlPayload{"", "Something went wrong."}
		_ = resultTemplate.Execute(w, payload)
		return
	}

	_ = resultTemplate.Execute(w, payload)
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

func (z Zap) register(router *mux.Router) {
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	router.HandleFunc("/", z.Home).Methods(http.MethodGet)
	router.HandleFunc("/r/{key}", z.Redirect).Methods(http.MethodGet)
	router.HandleFunc("/", z.Create).Methods(http.MethodPost)
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
