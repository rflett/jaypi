package main

import (
	"encoding/json"
	"errors"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// requestBody is the expected request body
type requestBody struct {
	Name string `json:"name"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var err error
	var status int

	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	// the user needs to be the group owner
	if ok, _ := services.UserIsGroupOwner(authContext.UserID, groupID); !ok {
		return services.ReturnError(errors.New("You have to be the group owner to do this"), http.StatusUnauthorized)
	}

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	err = json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// update
	group := types.Group{
		GroupID: groupID,
		OwnerID: authContext.UserID,
		Name:    reqBody.Name,
	}
	if status, err = group.Update(); err != nil {
		return services.ReturnError(err, status)
	}
	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
