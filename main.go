package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
)

var homeTemplate = template.Must(template.ParseFiles("index.html"))

func home(w http.ResponseWriter, r *http.Request) {
	err := homeTemplate.Execute(w, "index.html")
	if err != nil {
		w.Write([]byte("err"))
		return
	}

}

func create(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	fmt.Println(url)
	w.Write([]byte("You submitted: " + url))
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
