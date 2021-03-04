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
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &reqBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// join
	g, getGroupErr := services.GetGroupFromCode(reqBody.Code)
	if getGroupErr != nil {
		return events.APIGatewayProxyResponse{Body: getGroupErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	joinStatus, joinErr := g.AddUser(authContext.UserID)
	if joinErr != nil {
		return events.APIGatewayProxyResponse{Body: joinErr.Error(), StatusCode: joinStatus}, nil
	}

	// response
	responseBody, _ := json.Marshal(g)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(responseBody), StatusCode: http.StatusOK, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
