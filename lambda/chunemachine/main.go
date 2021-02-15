package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var (
	locale, _ = time.LoadLocation("Australia/Sydney")
	apiURL    = os.Getenv("API_URL")
)

type Artist struct {
	Entity       string        `json:"entity"`
	Arid         string        `json:"arid"`
	Name         string        `json:"name"`
	Artwork      []interface{} `json:"artwork"`
	Links        []interface{} `json:"links"`
	IsAustralian interface{}   `json:"is_australian"`
	Type         string        `json:"type"`
	Role         interface{}   `json:"role"`
}

type Recording struct {
	Entity      string        `json:"entity"`
	Arid        string        `json:"arid"`
	Title       string        `json:"title"`
	Metadata    string        `json:"metadata"`
	Description string        `json:"description"`
	Duration    int           `json:"duration"`
	Artists     []Artist      `json:"artists"`
	Releases    []interface{} `json:"releases"`
	Artwork     []interface{} `json:"artwork"`
	Links       []interface{} `json:"links"`
}

type Item struct {
	Entity     string      `json:"entity"`
	Count      int         `json:"count"`
	Arid       string      `json:"arid"`
	PlayedTime string      `json:"played_time"`
	ServiceID  string      `json:"service_id"`
	Recording  Recording   `json:"recording"`
	Release    interface{} `json:"release"`
}

type ResponseBody struct {
	Total  int    `json:"total"`
	Offset int    `json:"offset"`
	Items  []Item `json:"items"`
}

func sendToAPI(song *string) {
	// build the request body
	requestBody, err := json.Marshal(map[string]string{
		"songName": *song,
	})
	if err != nil {
		log.Logger.Error().Err(err).Msg("Issue building request body")
	}

	// send song to api
	resp, postErr := http.Post(apiURL, "application/json", bytes.NewBuffer(requestBody))

	if postErr != nil {
		log.Logger.Error().Err(postErr).Msg(fmt.Sprintf("Issue sending song %s to the API at %s", *song, apiURL))
	} else {
		// close response
		defer resp.Body.Close()
	}
}

func getSong() {
	log.Logger.Info().Msg("Checking latest song")

	toQueryParam := time.Now().In(locale).Format("2006-01-02T03:04:05")
	url := fmt.Sprintf("https://music.abcradio.net.au/api/v1/plays/search.json?station=triplej&limit=1&order=desc&qld&to=%s", toQueryParam)

	resp, err := http.Get(url)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Couldn't get latest song")
	}

	defer resp.Body.Close()

	response := ResponseBody{}
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	jsonErr := json.Unmarshal(bodyBytes, &response)
	if jsonErr != nil {
		log.Logger.Error().Err(jsonErr).Msg("Unable to Unmarshal JSON response")
	}

	log.Logger.Info().Msg(fmt.Sprintf("Current song is %s by %s", response.Items[0].Recording.Title, response.Items[0].Recording.Artists[0].Name))

	song := &response.Items[0].Recording.Title
	sendToAPI(song)
}

func main() {
}
