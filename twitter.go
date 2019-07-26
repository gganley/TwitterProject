package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

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

func getTweets(requestParam DataRequestParam) DataResponse {
	bearerToken := os.Getenv("BEARER_TOKEN")
	url := "https://api.twitter.com/1.1/tweets/search/30day/dev.json"

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

	err = json.NewDecoder(httpResponse.Body).Decode(&data)

	if err != nil {
		panic(err)
	}

	return data
}

func paginateTwitter(request DataRequestParam) (retVal []Tweet) {
	results := getTweets(request)

	for {
		for _, tweet := range results.Results {
			retVal = append(retVal, tweet)
		}

		if results.Next != "" {
			request.Next = results.Next
			results = getTweets(request)
		} else {
			break
		}
	}

	return
}
