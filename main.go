package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
)

func main() {
	r := chi.NewRouter()
	r.Mount("/", http.FileServer(http.Dir("client/dist")))
	r.Route("/api", func(r chi.Router) {
		r.Post("/search", PostSearchHandler)
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}
