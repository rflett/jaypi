package main

import (
	"encoding/json"
	"jjj.rflett.com/jjj-api/services"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// requestBody is the expected body of the create groupOld request
type requestBody struct {
	Code string `json:"code"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var err error
	var status int

	authContext := services.GetAuthorizerContext(request.RequestContext)

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	err = json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// join
	group, err := services.GetGroupFromCode(reqBody.Code)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}
	if status, err = group.AddUser(authContext.UserID); err != nil {
		return services.ReturnError(err, status)
	}
	return services.ReturnJSON(group, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
