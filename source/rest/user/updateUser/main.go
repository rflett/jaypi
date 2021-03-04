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
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &reqBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// get the user
	user := types.User{UserID: authContext.UserID}
	status, err := user.GetByUserID()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: status}, nil
	}

	// update the user
	user.NickName = &reqBody.NickName
	status, err = user.Update()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: status}, nil
	}

	// response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
