package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get songID from pathParameters
	songID := request.PathParameters["songId"]

	// create
	user := types.User{UserID: authContext.UserID}
	if status, err := user.RemoveVote(&songID); err != nil {
		return services.ReturnError(err, status)
	}
	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
