package main

import (
	"log"
	"math/rand"
	"net/http"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

var (
	homeTemplate   = template.Must(template.ParseFiles("template/index.html"))
	resultTemplate = template.Must(template.ParseFiles("template/result.html"))
)

const (
	charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

type UrlPayload struct {
	Original  string
	Shortened string
}

func randomString() string {
	// Define the set of characters that can be used in the random string.

	// Initialize a new random number generator with a seed value based on the current time.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create a new slice of bytes to hold the random string.
	b := make([]byte, 7)

	// Populate the slice with random characters from the charset.
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}

	// Return the string version of the byte slice.
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
	payload := UrlPayload{url, randomString()}
	err := homeTemplate.Execute(w, payload)
	if err != nil {
		return
	}
}

func main() {
	router := mux.NewRouter()
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	router.HandleFunc("/", home).Methods("GET")
	router.HandleFunc("/", create).Methods("POST")
	// router.HandleFunc("/result", result).Methods("GET")
	// router.HandleFunc("/{key}", go).Methods("GET")

	err := http.ListenAndServe(":9090", router)
	if err != nil {
		log.Fatalln("There's an error with the server", err)
	}
}
