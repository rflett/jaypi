package main

import (
	"jjj.rflett.com/jjj-api/types/group"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get ids from pathParameters
	groupID := request.PathParameters["groupId"]
	userID := request.PathParameters["userId"]

	// leave
	leaveStatus, leaveErr := group.Leave(userID, groupID)
	if leaveErr != nil {
		return events.APIGatewayProxyResponse{Body: leaveErr.Error(), StatusCode: leaveStatus}, nil
	}

	// response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
