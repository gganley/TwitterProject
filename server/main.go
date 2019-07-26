package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"google.golang.org/appengine"
)

func main() {
	r := chi.NewRouter()
	r.Get("/", RootHandler)
	http.Handle("/", r)
	appengine.Main()
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello world")
}
