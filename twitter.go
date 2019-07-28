package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var TWITTER_API_KEY = os.Getenv("TWITTER_API_KEY")
var TWITTER_API_SECRET = os.Getenv("TWITTER_API_SECRET")
var TWITTER_ACCESS_TOKEN = os.Getenv("TWITTER_ACCESS_TOKEN")
var TWITTER_ACCESS_TOKEN_SECRET = os.Getenv("TWITTER_ACCESS_TOKEN_SECRET")
var OAUTH_SIGNING_KEY = TWITTER_API_SECRET + TWITTER_ACCESS_TOKEN_SECRET
var BEARER_TOKEN = os.Getenv("BEARER_TOKEN")
var NPROC = 8

type Tweet struct {
	CreatedAt     string        `json:"created_at"`
	Id            int           `json:"id"`
	IdStr         string        `json:"id_str"`    // In the documentation it says to use this since a 64bit signed int conversion can get harry https://developer.twitter.com/en/docs/tweets/data-dictionary/overview/tweet-object
	Text          string        `json:"text"`      // 0-120 chars, not full 240. Full UTF-8, something to consider if we're counting chars https://github.com/twitter/twitter-text/blob/master/rb/lib/twitter-text/regex.rb
	Truncated     bool          `json:"truncated"` // compensates for new 240 char limit
	ExtendedTweet ExtendedTweet `json:"extended_tweet,omitempty"`
}

type ExtendedTweet struct {
	FullText string `json:"full_text"` // This is the full 240 chars, UTF-8
}

type DataRequestParam struct {
	Query      string `json:"query"`
	FromDate   string `json:"fromDate"`   // YYYYMMDDHHmm
	ToDate     string `json:"toDate"`     // Also YYYYMMDDHHmm
	MaxResults int    `json:"maxResults"` // Current system limit is 500 and 100 for sandbox. Defaults to 100
	Next       string `json:"next,omitempty"`
}

type DataResponse struct {
	Results           []Tweet          `json:"results"`
	Next              string           `json:"next,omitempty"` // When this is not present that means there are no more tweets to ask for
	RequestParameters DataRequestParam `json:"requestParameters,omitempty"`
}

type SavedSearch struct {
	TimeOfSearch time.Time   `json:"time_of_search"`
	SearchQuery  string      `json:"query"`
	TopWords     []WordCount `json:"top_words"`
}

type WordCount struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

func getTweetsFromFile(requestParam DataRequestParam) DataResponse {
	body, err := ioutil.ReadFile("ruby.json")

	if err != nil {
		panic(err)
	}

	var data DataResponse

	err = json.Unmarshal(body, &data)

	if err != nil {
		panic(err)
	}
	return data
}
func getTweets(requestParam DataRequestParam) DataResponse {
	bearerToken := os.Getenv("BEARER_TOKEN")
	url := "https://api.twitter.com/1.1/tweets/search/fullarchive/decfull.json"

	body, err := json.Marshal(&requestParam)

	if err != nil {
		panic(err)
	}

	client := http.Client{}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
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

func paginateTwitter(request DataRequestParam) <-chan []Tweet {
	out := make(chan []Tweet, 8)

	go func(request DataRequestParam) {
		log.Println("paginate goroutine")
		for i := 0; i < 10; i++ {
			results := getTweetsFromFile(request)
			time.Sleep(100 * time.Millisecond)
			log.Println(i)
			out <- results.Results
		}
		log.Println("Closing")
		close(out)
		// for {
		// 	log.Println("About to put on channel")
		// 	out <- results.Results
		// 	log.Println("Put on channel")
		// 	if results.Next != "" {
		// 		log.Println("More to go")
		// 		request.Next = results.Next
		// 		results = getTweets(request)
		// 	} else {
		// 		log.Println("Thats it closing")
		// 		close(out)
		// 		log.Println("closed")
		// 		return
		// 	}
		// }
	}(request)

	return out
}

func tallyTweets(in <-chan []Tweet) <-chan map[string]int {
	var wg sync.WaitGroup
	out := make(chan map[string]int, 8)

	work := func(tweets []Tweet) {
		defer wg.Done()
		log.Println("working")
		tally := make(map[string]int)
		time.Sleep(1000 * time.Millisecond)
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
		log.Println("Done working")
		out <- tally
	}

	for tweets := range in {
		log.Println("about to work")
		wg.Add(1)
		go work(tweets)
	}

	go func() {
		wg.Wait()
		log.Println("Closing tally")
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

	return wordCount
}
