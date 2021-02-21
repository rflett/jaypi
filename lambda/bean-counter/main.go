package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/dchest/uniuri"
	logger "jjj.rflett.com/jjj-api/log"
	"jjj.rflett.com/jjj-api/types/song"
	"jjj.rflett.com/jjj-api/types/user"
	"math/rand"
	"os"
	"time"
)

const (
	PrimaryKey     = "USER"
	SortKey        = "SONG"
	SecondaryIndex = "GSI1"
	MessageBatch   = 10
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	queue         = sqs.New(awsSession)
	table         = os.Getenv("JAYPI_TABLE")
	scorerQueue   = os.Getenv("SCORER_QUEUE")
)

type scorerMessageBody struct {
	Score  int    `json:"score"`
	UserID string `json:"userID"`
}

type beanMessageBody struct {
	SongID       string `json:"songID"`
	DelaySeconds *int64 `json:"delaySeconds"`
}

// queueForScorer takes a slice of userIDs and the score to give them and batches them onto SQS
func queueForScorer(score *int, userIDs []string) error {
	voterCount := len(userIDs)

	// loop through the userIDs and send them to SQS in batches of messageBatch
	for i := 0; i < voterCount; i += MessageBatch {
		j := i + MessageBatch
		if j > voterCount {
			j = voterCount
		}

		// create the batch of messageBatch entries
		var entries []*sqs.SendMessageBatchRequestEntry
		for _, userID := range userIDs[i:j] {
			mb, _ := json.Marshal(scorerMessageBody{UserID: userID, Score: *score})
			e := sqs.SendMessageBatchRequestEntry{
				Id:          aws.String(uniuri.NewLen(6)),
				MessageBody: aws.String(string(mb)),
			}
			entries = append(entries, &e)
		}

		// send the batch to SQS
		input := &sqs.SendMessageBatchInput{
			QueueUrl: &scorerQueue,
			Entries:  entries,
		}
		sendOutput, sendErr := queue.SendMessageBatch(input)
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

// calculateScore determines what score to give users for their song
func calculateScore(songPosition *int) *int {
	// TODO this is just a random score but we need to decide what to do here
	var score int
	rand.Seed(time.Now().UTC().UnixNano())
	score = *songPosition + rand.Intn(100)
	return &score
}

// getVoters returns the IDs of users who voted for a particular song
func getVoters(songID string) (voters []string, err error) {
	input := &dynamodb.QueryInput{
		ExpressionAttributeNames: map[string]*string{
			"#U": aws.String("userID"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", SortKey, songID)),
			},
			":pk": {
				S: aws.String(fmt.Sprintf("%s#", PrimaryKey)),
			},
		},
		TableName:              &table,
		IndexName:              aws.String(SecondaryIndex),
		KeyConditionExpression: aws.String("SK = :sk and begins_with(PK, :pk)"),
		ProjectionExpression:   aws.String("#U"),
		Limit:                  aws.Int64(1),
	}

	queryErr := db.QueryPages(input, func(page *dynamodb.QueryOutput, lastPage bool) bool {
		for _, item := range page.Items {
			voter := user.User{}
			unMarshErr := dynamodbattribute.UnmarshalMap(item, &voter)
			if unMarshErr != nil {
				logger.Log.Error().Err(unMarshErr).Msg("error unmarshalling item to user")
			}
			if voter.UserID != "" {
				voters = append(voters, voter.UserID)
			}
		}
		return !lastPage
	})

	if queryErr != nil {
		logger.Log.Error().Err(queryErr).Str("songID", songID).Msg("error getting voters for song")
		return []string{}, queryErr
	}

	return voters, nil
}

func HandleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	// unmarshall sqsEvent to messageBody
	mb := beanMessageBody{}
	jsonErr := json.Unmarshal([]byte(sqsEvent.Records[0].Body), &mb)
	if jsonErr != nil {
		logger.Log.Error().Err(jsonErr).Msg("Unable to unmarshal sqsEvent body to messageBody struct")
		return jsonErr
	}

	// get the full song details
	s := song.Song{SongID: mb.SongID}
	getSongErr := s.Get()
	if getSongErr != nil {
		logger.Log.Error().Err(getSongErr).Str("songID", mb.SongID).Msg("Unable to get the song from the table")
		return getSongErr
	}

	// calculate score for users who voted for this song
	if s.PlayedPosition == nil {
		playPosMissingErr := errors.New("playedPosition is nil")
		logger.Log.Error().Err(playPosMissingErr).Str("songID", mb.SongID).Msg("Song hasn't been played yet")
		return playPosMissingErr
	}
	score := calculateScore(s.PlayedPosition)

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

	// queue the voters and their score for the scorer function to process
	queueErr := queueForScorer(score, voters)
	if queueErr != nil {
		logger.Log.Error().Err(getVotersErr).Str("songID", mb.SongID).Msg("Unable to queue voters for scoring")
	}
	return queueErr
}

func main() {
	lambda.Start(HandleRequest)
}
