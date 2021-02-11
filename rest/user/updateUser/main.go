package main

import (
	"encoding/json"
	"jjj.rflett.com/jjj-api/types/user"
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

	// get userId from pathParameters
	userID := request.PathParameters["userId"]

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &reqBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// update
	u := user.User{
		UserID:   userID,
		NickName: reqBody.NickName,
	}
	updateStatus, updateErr := u.Update()
	if updateErr != nil {
		return events.APIGatewayProxyResponse{Body: updateErr.Error(), StatusCode: updateStatus}, nil
	}

	// response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
