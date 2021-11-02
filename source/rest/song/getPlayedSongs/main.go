package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

type responseBody struct {
	PlayedCount int          `json:"played_count"`
	Songs       []types.Song `json:"songs"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	startIndex := "0"
	if v, ok := request.QueryStringParameters["startIndex"]; ok {
		startIndex = v
	}

	numItems := "5"
	if v, ok := request.QueryStringParameters["numItems"]; ok {
		numItems = v
	}

	recentSongs, err := services.GetRecentlyPlayed(startIndex, numItems)
	if err != nil {
		return services.ReturnError(err, http.StatusInternalServerError)
	}

	currentPlayCount, err := services.GetCurrentPlayCount()
	if err != nil {
		return services.ReturnError(err, http.StatusInternalServerError)
	}

	// number of played songs is one less than the current play count
	currentPlayCount--

	// return the songs
	rb := responseBody{PlayedCount: currentPlayCount, Songs: recentSongs}
	return services.ReturnJSON(rb, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
