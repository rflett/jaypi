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
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// get attributes
	attributes, err := services.GetPlatformEndpointAttributes(reqBody.Endpoint)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// you can only delete your own endpoint
	if *attributes["CustomUserData"] != authContext.UserID {
		return services.ReturnError(errors.New("Cannot deregister endpoint not associated with this user"), http.StatusUnauthorized)
	}

	// you can only delete an endpoint that's still enabled
	if *attributes["Enabled"] != "true" {
		return services.ReturnError(errors.New("Cannot deregister disabled endpoint"), http.StatusBadRequest)
	}

	// delete
	platformEndpoint := types.PlatformEndpoint{Arn: reqBody.Endpoint, UserID: &authContext.UserID}
	err = platformEndpoint.Delete()
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}
	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
