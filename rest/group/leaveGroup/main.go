package main

import (
	"encoding/json"
	"jjj.rflett.com/jjj-api/types/group"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// requestBody is the expected body of the create groupOld request
type requestBody struct {
	UserID  string `json:"userID"`
	GroupID string `json:"groupID"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &reqBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// leave
	leaveStatus, leaveErr := group.Leave(reqBody.UserID, reqBody.GroupID)
	if leaveErr != nil {
		return events.APIGatewayProxyResponse{Body: leaveErr.Error(), StatusCode: leaveStatus}, nil
	}

	// response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
