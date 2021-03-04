package main

import (
	"encoding/json"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// requestBody is the expected body of the create groupOld request
type requestBody struct {
	Name string `json:"name"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &reqBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// create
	g := types.Group{
		OwnerID: authContext.UserID,
		Name:    reqBody.Name,
	}
	createStatus, createErr := g.Create()
	if createErr != nil {
		return events.APIGatewayProxyResponse{Body: createErr.Error(), StatusCode: createStatus}, nil
	}

	// response
	responseBody, _ := json.Marshal(g)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(responseBody), StatusCode: http.StatusCreated, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
