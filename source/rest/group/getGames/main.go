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

	// the user needs to be the group owner
	if ok, _ := services.UserIsGroupOwner(authContext.UserID, groupID); !ok {
		return services.ReturnError(errors.New("You have to be the group owner to do this"), http.StatusForbidden)
	}

	// get games
	group := types.Group{GroupID: groupID}
	games, err := group.GetGames()
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}
	return services.ReturnJSON(games, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
