package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"google.golang.org/appengine"
)

func main() {
	r := chi.NewRouter()
	r.Get("/", RootHandler)
	r.Route("/api", func(r chi.Router) {
		r.Post("/search", PostSearchHandler)
	})
	r.Mount("/", http.FileServer(http.Dir("client/dist")))
	http.Handle("/", r)
	// log.Fatal(http.ListenAndServe(":8080", r))
	appengine.Main()
}
