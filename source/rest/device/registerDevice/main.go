package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"os"
)

var (
	platforms = map[string]types.PlatformApp{
		types.SNSPlatformGoogle: {
			Arn:      os.Getenv("GOOGLE_PLATFORM_APP"),
			Platform: types.SNSPlatformGoogle,
		},
		types.SNSPlatformApple: {
			Arn:      os.Getenv("APPLE_PLATFORM_APP"),
			Platform: types.SNSPlatformApple,
		},
	}
)

// requestBody is the expected body of the create groupOld request
type requestBody struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
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

	// get app for the platform
	platformApp, exists := platforms[reqBody.Platform]
	if !exists {
		return services.ReturnError(errors.New(fmt.Sprintf("Unsupported platform %s", reqBody.Platform)), http.StatusBadRequest)
	}

	// get the user associated with the token
	platformEndpoint, err := platformApp.GetPlatformEndpointFromToken(&reqBody.Token)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// if the token is in use with an endpoint already then delete the endpoint
	if platformEndpoint != nil && platformEndpoint.UserID != authContext.UserID {
		err = platformEndpoint.Delete()
		if err != nil {
			return services.ReturnError(err, http.StatusBadRequest)
		}
	}

	// create the endpoint
	err = platformApp.CreatePlatformEndpoint(authContext.UserID, &reqBody.Token)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}
	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
