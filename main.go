package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
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
	projectID := os.Getenv("PROJECT_ID")

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v\n", err)
	}
	defer client.Close()
	timeOfExecution := time.Now()
	var response DataRequestParam

	err = json.NewDecoder(r.Body).Decode(&response)
	if err != nil {
		panic(err)
	}

	results := getTweets(response)

	tally := make(map[string]int)

	for {
		for _, tweet := range results.Results {
			var text string
			if tweet.ExtendedTweet.FullText != "" {
				text = tweet.ExtendedTweet.FullText
			} else {
				text = tweet.Text
			}
			for _, word := range strings.Fields(text) {
				tally[word] += 1
			}
		}

		if results.Next == "" {
			flippedTally := make(map[int][]string)
			final := make([]WordCount, 10)
			for key, value := range tally {
				flippedTally[value] = append(flippedTally[value], key)
			}

			keys := make([]int, 0, len(flippedTally))

			for k := range flippedTally {
				keys = append(keys, k)
			}

			// sort.Sort(sort.Reverse(sort.IntSlice(keys)))
			sort.Sort(sort.IntSlice(keys))
			i := 0
			for j := len(keys) - 1; j >= 0; j-- {
				for _, val := range flippedTally[keys[j]] {
					if i <= 9 {
						final[i] = WordCount{val, tally[val]}
						i++
					}
				}
			}
			_, _, err = client.Collection("search").Add(ctx, SavedSearch{timeOfExecution, response.Query, final})
			if err != nil {
				log.Fatalf("Failed to add search: %v\n", err)
			}
			return
		}
		response.Next = results.Next
		results = getTweets(response)
	}
}
