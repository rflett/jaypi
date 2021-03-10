package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/types"
)

func HandleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	// unmarshall sqsEvent to messageBody
	mb := types.CrierBody{}
	jsonErr := json.Unmarshal([]byte(sqsEvent.Records[0].Body), &mb)
	if jsonErr != nil {
		logger.Log.Error().Err(jsonErr).Msg("Unable to unmarshal sqsEvent body to messageBody struct")
		return jsonErr
	}

	// get user
	user := types.User{UserID: mb.UserID}
	endpoints, err := user.GetEndpoints()
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to get user")
		return err
	}

	// send notifications
	for _, endpoint := range *endpoints {
		endpoint.UserID = mb.UserID
		_ = endpoint.SendNotification(&mb.Notification)
	}

	// TODO publish sockets

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
