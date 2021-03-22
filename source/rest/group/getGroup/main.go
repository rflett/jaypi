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

	// check user is in the group
	if ok, _ := services.UserIsInGroup(authContext.UserID, groupID); !ok {
		return services.ReturnError(errors.New("You have to a member of the group to do this"), http.StatusForbidden)
	}

	// get group
	group := types.Group{GroupID: groupID}
	if status, err := group.Get(); err != nil {
		return services.ReturnError(err, status)
	}
	return services.ReturnJSON(group, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
