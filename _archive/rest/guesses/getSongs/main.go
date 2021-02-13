package main

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Song is a song in the hottest 100 that can be voted on
type Song struct {
	ID     string `json:"id"`
	Artist string `json:"artist"`
	Title  string `json:"title"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// these might go in redis or something as they won't change and will be queried heavliy
	songs := []Song{
		{
			ID:     "1",
			Artist: "Dream Theater",
			Title:  "Raise The Knife",
		},
		{
			ID:     "2",
			Artist: "Opeth",
			Title:  "When",
		},
	}

	// create and send the response
	body, _ := json.Marshal(songs)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: 200, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
