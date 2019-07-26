package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"google.golang.org/appengine"
)

func main() {
	r := chi.NewRouter()
	r.Get("/", RootHandler)
	r.Route("/api", func(r chi.Router) {
		r.Post("/search/{query}", PostSearchHandler)
	})
	r.Mount("/", http.FileServer(http.Dir("client/dist")))
	http.Handle("/", r)
	appengine.Main()
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("client/dist/index.html")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	body, err := ioutil.ReadAll(file)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintln(w, string(body))
}

func PostSearchHandler(w http.ResponseWriter, r *http.Request) {
	// searchQuery := r.Context().Value("query").(string)

	fmt.Fprintln(w, "Super cool api")
}
