package song

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	logger "jjj.rflett.com/jjj-api/log"
	"os"
	"time"
)

const (
	PrimaryKey = "SONG"
	SortKey    = "#PROFILE"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	table         = os.Getenv("JAYPI_TABLE")
)

// Song is a Song that a user can votes for
type Song struct {
	PK             string  `json:"-" dynamodbav:"PK"`
	SK             string  `json:"-" dynamodbav:"SK"`
	SongID         string  `json:"songID"`
	Name           string  `json:"name"`
	Artist         string  `json:"artist"`
	PlayedPosition *int    `json:"playedPosition"`
	PlayedAt       *string `json:"playedAt"`
	CreatedAt      *string `json:"createdAt"`
}

// Create the song
func (s *Song) Create() error {
	// set fields
	s.PK = fmt.Sprintf("%s#%s", PrimaryKey, s.SongID)
	s.SK = fmt.Sprintf("%s#%s", SortKey, s.SongID)
	createdAt := time.Now().UTC().Format(time.RFC3339)
	s.CreatedAt = &createdAt

	// create item
	av, _ := dynamodbattribute.MarshalMap(s)

	// create input
	input := &dynamodb.PutItemInput{
		Item:         av,
		ReturnValues: aws.String("NONE"),
		TableName:    aws.String(table),
	}

	// add to table
	_, err := db.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("Error adding song to table")
		return err
	}

	logger.Log.Info().Str("songID", s.SongID).Msg("Successfully added song to table")
	return nil
}

// Exists checks to see if the song exists in the table already and returns an error if it does
func (s *Song) Exists() error {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, s.SongID)),
			},
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", SortKey, s.SongID)),
			},
		},
		KeyConditionExpression: aws.String("SK = :sk and PK = :pk"),
		ProjectionExpression:   aws.String("songID"),
		TableName:              aws.String(table),
	}

	// query
	result, err := db.Query(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("error checking if song exists")
		return err
	}

	// song doesn't exist
	if len(result.Items) == 0 {
		logger.Log.Info().Str("songID", s.SongID).Msg("song does not exist")
		return nil
	}

	// code exists
	logger.Log.Info().Str("songID", s.SongID).Msg("song already exists")
	return errors.New("song already exists")
}
