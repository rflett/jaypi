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
	var err error
	var status int

	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get groupID and gameID from pathParameters
	groupID := request.PathParameters["groupId"]

	// the user needs to be the group owner
	if ok, _ := services.UserIsGroupOwner(authContext.UserID, groupID); !ok {
		return services.ReturnError(errors.New("You have to be the group owner to do delete the group"), http.StatusForbidden)
	}

	// get group
	group := types.Group{GroupID: groupID}
	if status, err = group.Get(); err != nil {
		return services.ReturnError(err, status)
	}

	users, err := group.GetMembers(false)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// the owner needs to be the last member of the group
	if len(users) != 1 {
		return services.ReturnError(
			errors.New("You have to be the last member of the group to delete it. Please nominate a new owner instead"),
			http.StatusForbidden,
		)
	}

	// delete the group
	if status, err = group.Delete(); err != nil {
		return services.ReturnError(err, status)
	}
	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
