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
)

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
	// Parse query information
	timeOfExecution := time.Now()
	var response DataRequestParam

	err := json.NewDecoder(r.Body).Decode(&response)
	if err != nil {
		panic(err)
	}

	// Get tweets
	results := paginateTwitter(response)

	tally := make(map[string]int)

	// Paginate
	for _, tweet := range results {
		var text string

		// So twitter decided to go from 140 char limit to a 240 char limit. Instead of versioning their API they
		// decided it would be easier to add an embedded field. This is easy to deal with in weakly typed language
		// but makes it annoying in Go
		if tweet.ExtendedTweet.FullText != "" {
			text = tweet.ExtendedTweet.FullText
		} else {
			text = tweet.Text
		}

		for _, word := range strings.Fields(text) {
			tally[word] += 1
		}
	}

	// Flip the tally. The value is the list of words that occur the key's number of times
	flippedTally := make(map[int][]string)

	for key, value := range tally {
		flippedTally[value] = append(flippedTally[value], key)
	}

	keys := make([]int, 0, len(flippedTally))

	for k := range flippedTally {
		keys = append(keys, k)
	}

	// I coudn't find a way to range through a reverse sorted int slice so I had to sort it then index through it reverse
	sort.Sort(sort.IntSlice(keys))

	// i keeps track of the fact that I'm only tracking 10 words, so if there are 11 words that occur the equal,
	// greatest amount then they will be the only ones included
	i := 0
	// This is initiated to 10 so that even if there are 9 words it won't mess with indexing
	wordCount := make([]WordCount, 10)

	for j := len(keys) - 1; j >= 0; j-- {
		for _, val := range flippedTally[keys[j]] {
			if i <= 9 {
				wordCount[i] = WordCount{val, tally[val]}
				i++
			}
		}
	}

	saveSearch := SavedSearch{timeOfExecution, response.Query, wordCount}
	projectID := os.Getenv("PROJECT_ID")

	// Open firestore
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v\n", err)
	}
	defer client.Close()

	_, _, err = client.Collection("search").Add(ctx, saveSearch)

	if err != nil {
		log.Fatalf("Failed to add search: %v\n", err)
	}

	data, err := json.MarshalIndent(saveSearch, "", "	")

	if err != nil {
		log.Fatalf("SaveSearch could not be marshalled %s", err)
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", data)
}
