package main

import (
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)
	user := types.User{UserID: authContext.UserID}
	if err := user.LeaveAllGroups(); err != nil {
		return services.ReturnError(err, http.StatusInternalServerError)
	}
	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
