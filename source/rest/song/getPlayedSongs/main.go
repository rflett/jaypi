package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

type responseBody struct {
	Songs []types.Song `json:"songs"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	recentSongs, err := services.GetRecentlyPlayed()
	if err != nil {
		return services.ReturnError(err, http.StatusInternalServerError)
	}

	// return the songs
	rb := responseBody{Songs: recentSongs}
	return services.ReturnJSON(rb, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
