package main

import (
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"strconv"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	// whether to get the members votes as well
	withVotes, _ := strconv.ParseBool(request.QueryStringParameters["withVotes"])

	// check user is in the group
	if ok, _ := services.UserIsInGroup(authContext.UserID, groupID); !ok {
		return services.ReturnError(errors.New("You have to a member of the group to do this"), http.StatusUnauthorized)
	}

	// get group
	group := types.Group{GroupID: groupID}
	users, err := group.GetMembers(withVotes)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}
	return services.ReturnJSON(users, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
