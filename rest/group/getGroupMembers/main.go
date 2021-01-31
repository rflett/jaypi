package main

import (
	"encoding/json"
	"jjj.rflett.com/jjj-api/types/group"
	"jjj.rflett.com/jjj-api/types/user"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	// get the group from the database
	group := group.Group{ID: groupID}
	err, getStatus := group.Get()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: getStatus}, nil
	}

	// get the group members from the database
	usersErr, usersStatus, users := user.GetAll(group.Members)
	if usersErr != nil {
		return events.APIGatewayProxyResponse{Body: usersErr.Error(), StatusCode: usersStatus}, nil
	}

	// create and send the response
	body, _ := json.Marshal(&users)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: 200, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
