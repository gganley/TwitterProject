package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Tweet is the listed structure accourding to link
type Tweet struct {
	CreatedAt     string        `json:"created_at"`
	ID            int           `json:"id"`
	IDStr         string        `json:"id_str"`    // In the documentation it says to use this since a 64bit signed int conversion can get harry https://developer.twitter.com/en/docs/tweets/data-dictionary/overview/tweet-object
	Text          string        `json:"text"`      // 0-120 chars, not full 240. Full UTF-8, something to consider if we're counting chars https://github.com/twitter/twitter-text/blob/master/rb/lib/twitter-text/regex.rb
	Truncated     bool          `json:"truncated"` // compensates for new 240 char limit
	ExtendedTweet ExtendedTweet `json:"extended_tweet,omitempty"`
}

// ExtendedTweet allows for the access of the `full_text` subfield that contains the 240 char
type ExtendedTweet struct {
	FullText string `json:"full_text"` // This is the full 240 chars, UTF-8
}

// DataRequestParam contains the information needed to perform a search request
type DataRequestParam struct {
	Query      string `json:"query"`
	FromDate   string `json:"fromDate"`   // YYYYMMDDHHmm
	ToDate     string `json:"toDate"`     // Also YYYYMMDDHHmm
	MaxResults int    `json:"maxResults"` // Current system limit is 500 and 100 for sandbox. Defaults to 100
	Next       string `json:"next,omitempty"`
}

// DataResponse is the response from twitter containing the data and the next request parameters to use
type DataResponse struct {
	Results           []Tweet          `json:"results"`
	Next              string           `json:"next,omitempty"` // When this is not present that means there are no more tweets to ask for
	RequestParameters DataRequestParam `json:"requestParameters,omitempty"`
}

// SavedSearch is the structure that is saved in the datastore
type SavedSearch struct {
	TimeOfSearch time.Time   `json:"time_of_search"`
	SearchQuery  string      `json:"query"`
	TopWords     []WordCount `json:"top_words"`
}

// WordCount is a simple tuple of the word and the number of times it occurs in the list of tweets
type WordCount struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

func getTweetsFromFile(requestParam DataRequestParam) DataResponse {
	body, err := ioutil.ReadFile(fmt.Sprintf("%s.json", requestParam.Query))

	if err != nil {
		_ = fmt.Errorf("could not read file %v", err)
		return DataResponse{[]Tweet{}, "", requestParam}
	}

	var data DataResponse

	err = json.Unmarshal(body, &data)

	if err != nil {
		panic(err)
	}
	return data
}
func getTweets(searchURL string, requestParam DataRequestParam) DataResponse {
	bearerToken := os.Getenv("BEARER_TOKEN")

	if bearerToken == "" {
		panic("Please set the BEARER_TOKEN env variable")
	}
	
	body, err := json.Marshal(&requestParam)

	if err != nil {
		panic(err)
	}

	client := http.Client{}

	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(body))

	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", bearerToken))
	req.Header.Set("Content-Type", "application/json")

	httpResponse, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	var data DataResponse

	body, err = ioutil.ReadAll(httpResponse.Body)

	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &data)

	if err != nil {
		panic(err)
	}

	return data
}

func paginateTwitter(searchURL string, request DataRequestParam) <-chan []Tweet {
	out := make(chan []Tweet, 8)

	go func(request DataRequestParam) {
		results := getTweets(searchURL, request)
		for {
			out <- results.Results
			if results.Next != "" {
				request.Next = results.Next
				results = getTweets(searchURL, request)
			} else {
				close(out)
				return
			}
		}
	}(request)

	return out
}

func paginateLocalFile(request DataRequestParam) <-chan []Tweet {
	out := make(chan []Tweet, 8)

	go func(request DataRequestParam) {
		results := getTweetsFromFile(request)
		out <- results.Results
		close(out)
	}(request)

	return out
}

func tallyTweets(in <-chan []Tweet) <-chan map[string]int {
	var wg sync.WaitGroup
	out := make(chan map[string]int, 8)

	work := func(tweets []Tweet) {
		defer wg.Done()
		tally := make(map[string]int)
		for _, tweet := range tweets {
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
				tally[word]++
			}
		}
		out <- tally
	}

	for tweets := range in {
		wg.Add(1)
		go work(tweets)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func aggragate(in <-chan map[string]int) []WordCount {
	tally := make(map[string]int)

	for shortTally := range in {
		for key, value := range shortTally {
			tally[key] += value
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
	sort.Ints(keys)

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

	return wordCount
}
