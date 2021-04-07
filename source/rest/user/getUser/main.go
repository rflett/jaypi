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

	// get userId from pathParameters
	userID := request.PathParameters["userId"]

	// get user
	user := types.User{UserID: userID}
	if status, err := user.GetByUserID(); err != nil {
		return services.ReturnError(err, status)
	}

	// users can get themselves without doing the group check
	if authContext.UserID != userID {
		// check auth user is in the same group as the user they're getting
		isInGroup := false
		for _, groupID := range *user.GroupIDs {
			if ok, _ := services.UserIsInGroup(authContext.UserID, groupID); ok {
				isInGroup = true
				break
			}
		}
		if !isInGroup {
			return services.ReturnError(errors.New("You have to a member of the group to do this"), http.StatusForbidden)
		}
	}

	// response
	return services.ReturnJSON(user, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
