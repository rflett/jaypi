package main

import (
	"encoding/json"
	"jjj.rflett.com/jjj-api/types/user"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// RequestBody is the expected body of the request
type RequestBody struct {
	FirstName  string `json:"firstName"`
	LastName string `json:"lastName"`
	NickName string `json:"nickName"`
	Email string `json:"email"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// unmarshall request body to RequestBody struct
	requestBody := RequestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &requestBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// create
	u := user.User{
		FirstName: requestBody.FirstName,
		LastName:  requestBody.LastName,
		NickName:  requestBody.NickName,
		Email:     &requestBody.Email,
	}
	createStatus, createErr := u.Create()
	if createErr != nil {
		return events.APIGatewayProxyResponse{Body: createErr.Error(), StatusCode: createStatus}, nil
	}

	// response
	responseBody, _ := json.Marshal(u)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(responseBody), StatusCode: http.StatusCreated, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
