package main

import (
	"encoding/json"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

type RequestBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
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
		return services.ReturnError(errors.New("You have to be the group owner to do this"), http.StatusForbidden)
	}

	// unmarshall request body to RequestBody struct
	reqBody := RequestBody{}
	err = json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// create
	game := types.Game{
		GroupID:     groupID,
		Name:        reqBody.Name,
		Description: reqBody.Description,
	}
	if status, err = game.Create(); err != nil {
		return services.ReturnError(err, status)
	}
	return services.ReturnJSON(game, http.StatusCreated)
}

func main() {
	lambda.Start(Handler)
}
