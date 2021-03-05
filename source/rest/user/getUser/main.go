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
		// check user is in the group
		if ok, _ := services.UserIsInGroup(authContext.UserID, *user.GroupID); !ok {
			return services.ReturnError(errors.New("You have to a member of the group to do this"), http.StatusUnauthorized)
		}
	}

	// response
	return services.ReturnJSON(user, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
