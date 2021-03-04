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

// requestBody is the expected body of the create groupOld request
type requestBody struct {
	Endpoint string `json:"endpoint"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	err := json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// get attributes
	attributes, err := services.GetPlatformEndpointAttributes(reqBody.Endpoint)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// you can only delete your own endpoint
	if *attributes["CustomUserData"] != authContext.UserID {
		return events.APIGatewayProxyResponse{Body: errors.New("endpoint must be owned by user").Error(), StatusCode: http.StatusUnauthorized}, nil
	}

	// you can only delete an endpoint that's still enabled
	if *attributes["Enabled"] != "true" {
		return events.APIGatewayProxyResponse{Body: errors.New("endpoint must be enabled").Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// delete
	platformEndpoint := types.PlatformEndpoint{Arn: reqBody.Endpoint, UserID: &authContext.UserID}
	err = platformEndpoint.Delete()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
