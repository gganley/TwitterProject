package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
)

// APISearchHandler allows for searching using either the 30Day search or the Full Archive search
func APISearchHandler(searchURL string, w http.ResponseWriter, r *http.Request) {
	// Parse query information
	timeOfExecution := time.Now()
	var response DataRequestParam

	err := json.NewDecoder(r.Body).Decode(&response)
	if err != nil {
		panic(err)
	}

	in := paginateTwitter(searchURL, response)

	tallyed := tallyTweets(in)

	wordCount := aggragate(tallyed)

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

// SearchFullArchiveHandler is given to /api/search/fullarchive
func SearchFullArchiveHandler(w http.ResponseWriter, r *http.Request) {
	APISearchHandler("https://api.twitter.com/1.1/tweets/search/fullarchive/decfull.json", w, r)
}

// Search30DayHandler is given to /api/search/30day
func Search30DayHandler(w http.ResponseWriter, r *http.Request) {
	APISearchHandler("https://api.twitter.com/1.1/tweets/search/30day/dev.json", w, r)
}

// SearchLocalFileHandler is a complete hack that I had to rought on this earth when I ran out of twitter API requests
func SearchLocalFileHandler(w http.ResponseWriter, r *http.Request) {
	timeOfExecution := time.Now()
	var response DataRequestParam

	err := json.NewDecoder(r.Body).Decode(&response)
	if err != nil {
		panic(err)
	}

	in := paginateLocalFile(response)

	tallyed := tallyTweets(in)

	wordCount := aggragate(tallyed)

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
