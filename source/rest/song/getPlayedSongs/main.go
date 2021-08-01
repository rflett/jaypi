package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"net/http"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	recentSongs, err := services.GetRecentlyPlayed()
	if err != nil {
		return services.ReturnError(err, http.StatusInternalServerError)
	}
	return services.ReturnJSON(recentSongs, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
