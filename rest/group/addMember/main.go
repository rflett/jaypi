package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/types/group"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// unmarshall request body to RequestDto struct
	userID := request.PathParameters["userId"]
	groupID := request.PathParameters["groupId"]

	// add user to group
	g := group.Group{ID: groupID}
	err, addStatus := g.AddMember(userID)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: addStatus}, nil
	}

	// create and send the response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: addStatus}, nil
}

func main() {
	lambda.Start(Handler)
}
