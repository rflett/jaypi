package main

import (
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]
	userID := request.PathParameters["userId"]

	// if you're removing someone else you need to be the group owner
	if authContext.UserID != userID {
		if ok, _ := services.UserIsGroupOwner(authContext.UserID, groupID); !ok {
			return services.ReturnError(errors.New("You have to be the group owner to do this"), http.StatusForbidden)
		}
	} else {
		if ok, _ := services.UserIsInGroup(userID, groupID); !ok {
			return services.ReturnError(errors.New("You have to a member of the group to do this"), http.StatusForbidden)
		}
	}

	// get the user
	user := types.User{UserID: userID}

	// leave their current group
	if status, err := user.LeaveGroup(groupID); err != nil {
		return services.ReturnError(err, status)
	}

	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
