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

// requestBody is the expected body of the nominate user request
type requestBody struct {
	UserID  string `json:"userID"`
	GroupID string `json:"groupID"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var err error
	var status int

	authContext := services.GetAuthorizerContext(request.RequestContext)

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	err = json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// the user needs to be the group owner
	if ok, _ := services.UserIsGroupOwner(authContext.UserID, reqBody.GroupID); !ok {
		return services.ReturnError(errors.New("You have to be the group owner to do this"), http.StatusForbidden)
	}

	// the user must be in the group
	if ok, _ := services.UserIsInGroup(reqBody.UserID, reqBody.GroupID); !ok {
		return services.ReturnError(errors.New("This user is not in the group"), http.StatusForbidden)
	}

	// update group owner
	group := types.Group{
		GroupID: reqBody.GroupID,
	}
	if status, err = group.NominateOwner(reqBody.UserID); err != nil {
		return services.ReturnError(err, status)
	}
	return services.ReturnJSON(group, http.StatusCreated)
}

func main() {
	lambda.Start(Handler)
}
