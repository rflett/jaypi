package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/types/group"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	// get group
	g, responseStatus, err := group.Get(groupID)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: responseStatus}, nil
	}

	// response
	responseBody, _ := json.Marshal(g)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(responseBody), StatusCode: responseStatus, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
