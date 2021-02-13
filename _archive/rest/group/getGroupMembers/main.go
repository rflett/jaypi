package main

import (
	"encoding/json"
	"jjj.rflett.com/jjj-api/_archive/types/groupOld"
	"jjj.rflett.com/jjj-api/types/user"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	// get the groupOld from the database
	group := groupOld.Group{ID: groupID}
	err, getStatus := group.Get()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: getStatus}, nil
	}

	// get the groupOld members from the database
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
