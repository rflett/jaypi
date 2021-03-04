package main

import (
	"encoding/json"
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
	status, err := user.GetByUserID()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: status}, nil
	}

	// users can get themselves without doing the group check
	if authContext.UserID != userID {
		// check user is in the group
		if ok, _ := services.UserIsInGroup(authContext.UserID, *user.GroupID); !ok {
			return events.APIGatewayProxyResponse{Body: "You have to a member of the group to do this.", StatusCode: http.StatusUnauthorized}, nil
		}
	}

	// response
	responseBody, _ := json.Marshal(user)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(responseBody), StatusCode: status, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
