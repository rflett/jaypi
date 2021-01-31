package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	logger "jjj.rflett.com/jjj-api/log"
	"jjj.rflett.com/jjj-api/types/group"
	"net/http"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// get groupID and userID from pathParameters
	groupID := request.PathParameters["groupId"]
	userID := request.PathParameters["userId"]

	// get group
	g := group.Group{ID: groupID}
	getErr, getStatus := g.Get()
	if getErr != nil {
		return events.APIGatewayProxyResponse{Body: getErr.Error(), StatusCode: getStatus}, nil
	}

	// don't allow removing the owner from the members
	if userID == g.Owner {
		ownerMemberError := errors.New("cannot remove group owner from group members")
		logger.Log.Info().Str("groupID", groupID).Str("groupMember", userID).Msg(fmt.Sprintf("%s", ownerMemberError))
		return events.APIGatewayProxyResponse{
			Body: ownerMemberError.Error(),
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	// remove userID from group
	err, removeStatus := g.RemoveMember(userID)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: removeStatus}, nil
	}

	// create and send the response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: removeStatus}, nil
}

func main() {
	lambda.Start(Handler)
}
