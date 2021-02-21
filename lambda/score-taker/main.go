package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	logger "jjj.rflett.com/jjj-api/log"
	"jjj.rflett.com/jjj-api/types/user"
)

type messageBody struct {
	Score  int    `json:"score"`
	UserID string `json:"userID"`
}

func HandleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	// unmarshall sqsEvent to messageBody
	mb := messageBody{}
	jsonErr := json.Unmarshal([]byte(sqsEvent.Records[0].Body), &mb)
	if jsonErr != nil {
		logger.Log.Error().Err(jsonErr).Msg("Unable to unmarshal sqsEvent body to messageBody struct")
		return jsonErr
	}

	// append score to users score
	u := user.User{UserID: mb.UserID}
	err := u.UpdatePoints(mb.Score)
	if err == nil {
		logger.Log.Info().Str("userID", u.UserID).Msg(fmt.Sprintf("Added %d points to user", mb.Score))
	}
	return err
}

func main() {
	lambda.Start(HandleRequest)
}
