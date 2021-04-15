package ping

import "github.com/aws/aws-lambda-go/events"

// Literally does nothing, just here to have the auth handler in front of to validate JWT's for the frontend
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{StatusCode: 204}, nil
}
