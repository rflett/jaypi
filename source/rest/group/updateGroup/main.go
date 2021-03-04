package main

import (
	"encoding/json"
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
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	// the user needs to be the group owner
	if ok, _ := services.UserIsGroupOwner(authContext.UserID, groupID); !ok {
		return events.APIGatewayProxyResponse{Body: "You have to be the group owner to do this.", StatusCode: http.StatusUnauthorized}, nil
	}

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &reqBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// update
	g := types.Group{
		GroupID: groupID,
		OwnerID: authContext.UserID,
		Name:    reqBody.Name,
	}
	updateStatus, updateErr := g.Update()
	if updateErr != nil {
		return events.APIGatewayProxyResponse{Body: updateErr.Error(), StatusCode: updateStatus}, nil
	}

	// response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
