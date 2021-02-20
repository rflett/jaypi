package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/types/groupCode"
	"net/http"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	// get group QR code
	qr, err := groupCode.QR(groupID)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusInternalServerError}, nil
	}

	// response
	headers := map[string]string{"Content-Type": "application/text"}
	return events.APIGatewayProxyResponse{Body: qr, StatusCode: http.StatusOK, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
