package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

type key struct{}

var contextKey = &key{}

// requestBody is the expected body of the create groupOld request
type requestBody struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// Handler is our handle on life
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authInfo, _ := fromContext(ctx)

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &reqBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	switch reqBody.Platform {
	case "android":
		// do android things
	case "ios":
		// do ios things
	default:
		return events.APIGatewayProxyResponse{Body: errors.New("unsupported platform").Error(), StatusCode: http.StatusBadRequest}, nil
	}
}

func main() {
	lambda.Start(Handler)
}

// FromContext returns the LambdaContext value stored in ctx, if any.
// This is basically stolen from github.com/aws/aws-lambda-go/lambdacontext so we can parse our own custom context
func fromContext(ctx context.Context) (*types.AuthorizerContext, bool) {
	lc, ok := ctx.Value(contextKey).(*types.AuthorizerContext)
	return lc, ok
}
