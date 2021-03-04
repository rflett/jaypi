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
	_, err := user.GetByUserID()
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to get user")
		return err
	}

	// send notification to android endpoints
	if user.AndroidEndpoints != nil {
		for _, arn := range *user.AndroidEndpoints {
			endpoint := types.PlatformEndpoint{UserID: &user.UserID, Arn: arn}
			_ = endpoint.SendAndroidNotification(mb.Notification)
		}
	}

	// send notification to ios endpoints
	if user.IOSEndpoints != nil {
		for _, arn := range *user.IOSEndpoints {
			endpoint := types.PlatformEndpoint{UserID: &user.UserID, Arn: arn}
			_ = endpoint.SendAppleNotification(mb.Notification)
		}
	}

	// TODO publish sockets

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
