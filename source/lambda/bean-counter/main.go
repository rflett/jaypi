package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/dchest/uniuri"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/types"
	"os"
)

const MessageBatch = 10

var scorerQueue = os.Getenv("SCORER_QUEUE")

// queueForScorer takes a slice of userIDs and the score to give them and batches them onto SQS
func queueForScorer(points *int, userIDs []string) error {
	voterCount := len(userIDs)

	// loop through the userIDs and send them to SQS in batches of messageBatch
	for i := 0; i < voterCount; i += MessageBatch {
		j := i + MessageBatch
		if j > voterCount {
			j = voterCount
		}

		// create the batch of messageBatch entries
		var entries []sqsTypes.SendMessageBatchRequestEntry
		for _, userID := range userIDs[i:j] {
			messageBody, _ := json.Marshal(types.ScoreTakerBody{UserID: userID, Points: *points})
			entry := sqsTypes.SendMessageBatchRequestEntry{
				Id:          aws.String(uniuri.NewLen(6)),
				MessageBody: aws.String(string(messageBody)),
			}
			entries = append(entries, entry)
		}

		// send the batch to SQS
		input := &sqs.SendMessageBatchInput{
			QueueUrl: &scorerQueue,
			Entries:  entries,
		}
		sendOutput, sendErr := clients.SQSClient.SendMessageBatch(context.TODO(), input)
		if sendErr != nil {
			logger.Log.Error().Err(sendErr).Msg("Unable to send message batch to SQS")
			return sendErr
		}

		// check send results
		logger.Log.Info().Msg(fmt.Sprintf("Successfully put %d messages on the queue", len(sendOutput.Successful)))

		if len(sendOutput.Failed) > 0 {
			logger.Log.Warn().Msg(fmt.Sprintf("Failed to put %d messages on the queue", len(sendOutput.Failed)))
			for _, failedMessage := range sendOutput.Failed {
				logger.Log.Warn().Str("id", *failedMessage.Id).Msg(*failedMessage.Message)
			}
		}
	}
	return nil
}

// getVoters returns the IDs of users who voted for a particular song
func getVoters(songID string) (voters []string, err error) {
	pkCondition := expression.Key(types.PartitionKey).BeginsWith(fmt.Sprintf("%s#", types.UserPartitionKey))
	skCondition := expression.Key(types.SortKey).Equal(expression.Value(fmt.Sprintf("%s#%s", types.SongPartitionKey, songID)))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("UserID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for getVoters func")
	}

	// input
	input := &dynamodb.QueryInput{
		TableName:                 &types.DynamoTable,
		IndexName:                 aws.String(types.GSI),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	paginator := dynamodb.NewQueryPaginator(clients.DynamoClient, input)

	for paginator.HasMorePages() {
		page, pageErr := paginator.NextPage(context.TODO())
		if pageErr != nil {
			logger.Log.Error().Err(pageErr).Msg("error getting NextPage from getVoters paginator")
			break
		}

		var theseVoters []types.User
		marshalErr := attributevalue.UnmarshalListOfMaps(page.Items, &theseVoters)
		if marshalErr != nil {
			logger.Log.Error().Err(marshalErr).Msg("error unmarshalling voters to slice of users")
			break
		}

		for _, voter := range theseVoters {
			if voter.UserID != "" {
				voters = append(voters, voter.UserID)
			}
		}
	}

	return voters, nil
}

func HandleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	// unmarshall sqsEvent to messageBody
	mb := types.BeanCounterBody{}
	jsonErr := json.Unmarshal([]byte(sqsEvent.Records[0].Body), &mb)
	if jsonErr != nil {
		logger.Log.Error().Err(jsonErr).Msg("Unable to unmarshal sqsEvent body to messageBody struct")
		return jsonErr
	}

	// get the full song details
	s := types.Song{SongID: mb.SongID}
	getSongErr := s.Get()
	if getSongErr != nil {
		logger.Log.Error().Err(getSongErr).Str("songID", mb.SongID).Msg("Unable to get the song from the table")
		return getSongErr
	}

	// calculate points for users who voted for this song
	if s.PlayedPosition == nil {
		playPosMissingErr := errors.New("playedPosition is nil")
		logger.Log.Error().Err(playPosMissingErr).Str("songID", mb.SongID).Msg("Song hasn't been played yet")
		return playPosMissingErr
	}
	points := s.PlayedPosition

	// find the voters of this song
	voters, getVotersErr := getVoters(s.SongID)
	if getVotersErr != nil {
		logger.Log.Error().Err(getVotersErr).Str("songID", mb.SongID).Msg("Unable to get voters for the song")
		return getVotersErr
	}
	if len(voters) == 0 {
		logger.Log.Info().Str("songID", mb.SongID).Msg("No-one voted for this song")
		return nil
	}

	// queue the voters and their points for the scorer function to process
	queueErr := queueForScorer(points, voters)
	if queueErr != nil {
		logger.Log.Error().Err(getVotersErr).Str("songID", mb.SongID).Msg("Unable to queue voters for scoring")
	}
	return queueErr
}

func main() {
	lambda.Start(HandleRequest)
}
