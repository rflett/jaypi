package main

import (
	"encoding/json"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// requestBody is the expected body of the update user request
type requestBody struct {
	NickName string `json:"nickName"`
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

	// get the user
	user := types.User{UserID: authContext.UserID}
	if status, err = user.GetByUserID(); err != nil {
		return services.ReturnError(err, status)
	}

	// update the user
	user.NickName = &reqBody.NickName
	if status, err = user.Update(); err != nil {
		return services.ReturnError(err, status)
	}
	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
