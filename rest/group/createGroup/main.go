package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"jjj.rflett.com/jjj-api/types/group"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// BodyRequest is the expected body of the create group request
type BodyRequest struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// unmarshall request body to BodyRequest struct
	bodyRequest := BodyRequest{}
	err := json.Unmarshal([]byte(request.Body), &bodyRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// create
	g := group.Group{
		ID:    uuid.NewString(),
		Name:  bodyRequest.Name,
		Owner: bodyRequest.Owner,
	}
	createErr, createStatus := g.Add()
	if createErr != nil {
		return events.APIGatewayProxyResponse{Body: createErr.Error(), StatusCode: createStatus}, nil
	}

	// create and send the response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
