package main

import (
	"github.com/google/uuid"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/services"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	providerName := request.PathParameters["provider"]
	provider, err := services.GetOauthProvider(providerName)

	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to retrieve an oauth provider by name")
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// Have the provider, now redirect them to the login URL.
	// state string is for csrf but can't really occur with modern oauth, but if required implement
	stateStr := uuid.NewString()
	headers := map[string]string{"Location": provider.AuthCodeURL(stateStr)}
	return events.APIGatewayProxyResponse{Body: nil, StatusCode: http.StatusTemporaryRedirect, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
