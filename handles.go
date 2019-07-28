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

func PostSearchHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query information
	timeOfExecution := time.Now()
	var response DataRequestParam

	err := json.NewDecoder(r.Body).Decode(&response)
	if err != nil {
		panic(err)
	}

	in := paginateTwitter(response)

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
