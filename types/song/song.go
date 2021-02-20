package song

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	logger "jjj.rflett.com/jjj-api/log"
	"jjj.rflett.com/jjj-api/types/jjj"
	"jjj.rflett.com/jjj-api/types/playCount"
	"regexp"
	"time"
)

const (
	PrimaryKey = "SONG"
	SortKey    = "#PROFILE"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	table         = "jaypi"
)

// Song is a Song that a user can votes for
type Song struct {
	PK             string             `json:"-" dynamodbav:"PK"`
	SK             string             `json:"-" dynamodbav:"SK"`
	SongID         string             `json:"songID"`
	Name           string             `json:"name"`
	Album          string             `json:"album"`
	Artist         string             `json:"artist"`
	Artwork        *[]jjj.ArtworkSize `json:"artwork"`
	PlayedPosition *int               `json:"playedPosition"`
	PlayedAt       *string            `json:"playedAt"`
	CreatedAt      *string            `json:"createdAt"`
}

// SearchString should be used in spotify requests to search for the song
func (s *Song) SearchString() string {
	// only allow letters, numbers and spaces in search string
	reg, _ := regexp.Compile("[^a-zA-Z0-9 ]+")
	return reg.ReplaceAllString(fmt.Sprintf("%s %s", s.Name, s.Artist), "")
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
func (s *Song) Exists() (bool, error) {
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
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				return false, nil
			}
			logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("Error checking if song exists in table")
			return false, err
		} else {
			logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("Error checking if song exists in table")
			return false, err
		}
	}

	// song doesn't exist
	if len(result.Items) == 0 {
		logger.Log.Info().Str("songID", s.SongID).Msg("Song does not exist in table")
		return false, nil
	}

	// song exists
	logger.Log.Info().Str("songID", s.SongID).Msg("Song already exists in table")
	return true, nil
}

// Update the song
func (s *Song) Played() error {
	pk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, s.SongID)),
	}
	sk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", SortKey, s.SongID)),
	}

	// get current played count
	currentPlayCount, playCountErr := playCount.GetCurrentPlayCount()
	if playCountErr != nil {
		return playCountErr
	}

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#PA": aws.String("playedAt"),
			"#PP": aws.String("playedPosition"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": &pk,
			":sk": &sk,
			":pa": {
				S: aws.String(*s.PlayedAt),
			},
			":pp": {
				N: currentPlayCount,
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": &pk,
			"SK": &sk,
		},
		ReturnValues:        aws.String("NONE"),
		TableName:           aws.String(table),
		ConditionExpression: aws.String("PK = :pk and SK = :sk"),
		UpdateExpression:    aws.String("SET #PA = :pa, #PP = :pp"),
	}

	_, err := db.UpdateItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("Error updating song")
		return err
	}

	playCount.IncrementPlayCount()
	return nil
}
