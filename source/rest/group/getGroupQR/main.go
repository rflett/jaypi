package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	// check user is in the group
	if ok, _ := services.UserIsInGroup(authContext.UserID, groupID); !ok {
		return events.APIGatewayProxyResponse{Body: "You have to a member of the group to do this.", StatusCode: http.StatusUnauthorized}, nil
	}

	// get group QR code
	g := types.Group{GroupID: groupID}
	qr, err := g.GetQRCode()
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
