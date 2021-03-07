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

	// get the user
	user := types.User{UserID: authContext.UserID}
	if status, err := user.GetByUserID(); err != nil {
		return services.ReturnError(err, status)
	}

	if user.GroupID == nil {
		return services.ReturnError(errors.New("You're not a member of any groups"), http.StatusBadRequest)
	}

	// leave their current group
	if status, err := user.LeaveGroup(); err != nil {
		return services.ReturnError(err, status)
	}

	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
