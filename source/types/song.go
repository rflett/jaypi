package types

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/types/jjj"
	"regexp"
	"time"
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
	ViewPosition   *int				  `json:"viewPosition" dynamodbav:"position"`
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
	s.PK = fmt.Sprintf("%s#%s", SongPrimaryKey, s.SongID)
	s.SK = fmt.Sprintf("%s#%s", SongSortKey, s.SongID)
	createdAt := time.Now().UTC().Format(time.RFC3339)
	s.CreatedAt = &createdAt

	// create item
	av, _ := dynamodbattribute.MarshalMap(s)

	// create input
	input := &dynamodb.PutItemInput{
		Item:         av,
		ReturnValues: aws.String("NONE"),
		TableName:    &clients.DynamoTable,
	}

	// add to table
	_, err := clients.DynamoClient.PutItem(input)

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
				S: aws.String(fmt.Sprintf("%s#%s", SongPrimaryKey, s.SongID)),
			},
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", SongSortKey, s.SongID)),
			},
		},
		KeyConditionExpression: aws.String("SK = :sk and PK = :pk"),
		ProjectionExpression:   aws.String("songID"),
		TableName:              &clients.DynamoTable,
	}

	// query
	result, err := clients.DynamoClient.Query(input)

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

// Played marks the song as played and records its play time and position
func (s *Song) Played() error {
	pk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", SongPrimaryKey, s.SongID)),
	}
	sk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", SongSortKey, s.SongID)),
	}

	// get current played count
	currentPlayCount, playCountErr := getCurrentPlayCount()
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
				S: s.PlayedAt,
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
		TableName:           &clients.DynamoTable,
		ConditionExpression: aws.String("PK = :pk and SK = :sk"),
		UpdateExpression:    aws.String("SET #PA = :pa, #PP = :pp"),
	}

	_, err := clients.DynamoClient.UpdateItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("Error updating song")
		return err
	}

	incrementPlayCount()
	return nil
}

// Get the song from the table
func (s *Song) Get() error {
	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", SongPrimaryKey, s.SongID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", SongSortKey, s.SongID)),
			},
		},
		TableName: &clients.DynamoTable,
	}

	// getItem
	result, err := clients.DynamoClient.GetItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("error getting song from table")
		return err
	}

	if len(result.Item) == 0 {
		return errors.New("unable to find song in table")
	}

	// unmarshal item into struct
	err = dynamodbattribute.UnmarshalMap(result.Item, &s)
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("failed to unmarshal dynamo item to group")
	}
	return nil
}

// getCurrentPlayCount looks up the current playCount item and returns its value
func getCurrentPlayCount() (*string, error) {
	input := &dynamodb.QueryInput{
		ExpressionAttributeNames: map[string]*string{
			"#V": aws.String("value"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(PlayCountPrimaryKey),
			},
			":sk": {
				S: aws.String(PlayCountSortKey),
			},
		},
		KeyConditionExpression: aws.String("SK = :sk and PK = :pk"),
		ProjectionExpression:   aws.String("#V"),
		TableName:              &clients.DynamoTable,
	}
	result, err := clients.DynamoClient.Query(input)
	if err != nil || *result.Count == 0 {
		logger.Log.Error().Err(err).Msg("Unable to get the latest song position")
		return aws.String("0"), err
	}

	var pc = PlayCount{}
	unmarshalErr := dynamodbattribute.UnmarshalMap(result.Items[0], &pc)
	if unmarshalErr != nil {
		logger.Log.Error().Err(unmarshalErr).Msg("Unable to unmarshall query result to playCount")
		return aws.String("0"), unmarshalErr
	}
	return pc.Value, nil
}

// incrementPlayCount increments the current playCount value
func incrementPlayCount() {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#V": aws.String("value"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":inc": {
				N: aws.String("1"),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(PlayCountPrimaryKey),
			},
			"SK": {
				S: aws.String(PlayCountSortKey),
			},
		},
		ReturnValues:     aws.String("NONE"),
		TableName:        &clients.DynamoTable,
		UpdateExpression: aws.String("ADD #V :inc"),
	}
	_, err := clients.DynamoClient.UpdateItem(input)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to increment the latest song position")
	}
}
