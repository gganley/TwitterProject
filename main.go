package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"google.golang.org/appengine"
)

func main() {
	r := chi.NewRouter()
	r.Mount("/", http.FileServer(http.Dir("client/dist")))
	r.Route("/api", func(r chi.Router) {
		r.Post("/search", PostSearchHandler)
	})

	http.Handle("/", r)
	appengine.Main()
}
