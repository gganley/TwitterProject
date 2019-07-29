package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi"
)

var PORT = os.Getenv("PORT")

func main() {
	r := chi.NewRouter()
	r.Mount("/", http.FileServer(http.Dir("client/dist")))
	r.Route("/api/search", func(r chi.Router) {
		r.Post("/fullarchive", SearchFullArchiveHandler)
		r.Post("/30day", Search30DayHandler)
		r.Post("/file", SearchLocalFileHandler)
	})

	log.Fatal(http.ListenAndServe(":"+PORT, r))
}
