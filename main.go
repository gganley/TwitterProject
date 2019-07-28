package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
)

func main() {
	r := chi.NewRouter()
	r.Mount("/", http.FileServer(http.Dir("client/dist")))
	r.Route("/api/search", func(r chi.Router) {
		r.Post("/fullarchive", SearchFullArchiveHandler)
		r.Post("/30day", Search30DayHandler)
		r.Post("/file", SearchLocalFileHandler)
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}
