package main

import (
	"jjj.rflett.com/jjj-api/types"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get userId from pathParameters
	userID := request.PathParameters["userId"]
	songID := request.PathParameters["songId"]

	// create
	u := types.User{UserID: userID}
	deleteStatus, deleteErr := u.RemoveVote(&songID)
	if deleteErr != nil {
		return events.APIGatewayProxyResponse{Body: deleteErr.Error(), StatusCode: deleteStatus}, nil
	}

	// response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
