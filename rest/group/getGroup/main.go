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

	// get the group from the database
	newGroup := group.Group{ID: groupID}
	err, getStatus := newGroup.Get()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: getStatus}, nil
	}

	// create and send the response
	body, _ := json.Marshal(newGroup)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: getStatus, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
