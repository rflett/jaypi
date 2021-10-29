package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"net/http"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	if err := authContext.IsAdmin(); err != nil {
		return services.ReturnError(err, http.StatusForbidden)
	}

	services.PurgeSongs()
	services.SetPlayCount("1")

	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
