package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/types"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get userId from pathParameters
	userID := request.PathParameters["userId"]

	// get user
	u := types.User{UserID: userID}
	status, err := u.GetByUserID()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: status}, nil
	}

	// response
	responseBody, _ := json.Marshal(u)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(responseBody), StatusCode: status, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
