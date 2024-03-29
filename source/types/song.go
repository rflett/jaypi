package types

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/types/jjj"
	"regexp"
	"strconv"
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
	Rank           *int               `json:"rank" dynamodbav:"-"`
	PlayedPosition *int               `json:"playedPosition"`
	PlayedAt       *string            `json:"playedAt"`
	CreatedAt      *string            `json:"createdAt"`
}

// return the partition key value for a song
func (s *Song) PKVal() string {
	return fmt.Sprintf("%s#%s", SongPartitionKey, s.SongID)
}

// return the sort key value for a song
func (s *Song) SKVal() string {
	return fmt.Sprintf("%s#%s", SongSortKey, s.SongID)
}

// SearchString should be used in spotify requests to search for the song
func (s *Song) SearchString() string {
	// only allow letters, numbers and spaces in search string
	reg, _ := regexp.Compile("[^a-zA-Z0-9 ]+")
	return reg.ReplaceAllString(fmt.Sprintf("%s %s", s.Name, s.Artist), "")
}

// Delete the song
func (s *Song) Delete() error {
	// input
	input := &dynamodb.DeleteItemInput{
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: s.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: s.SKVal()},
		},
		TableName: &DynamoTable,
	}

	// delete from table
	_, err := clients.DynamoClient.DeleteItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("error deleting song")
	}

	return err
}

// Create the song
func (s *Song) Create() error {
	// set fields, everything else should come from the front-end (artwork etc.)
	s.PK = s.PKVal()
	s.SK = s.SKVal()
	createdAt := time.Now().UTC().Format(time.RFC3339)
	s.CreatedAt = &createdAt

	// create item
	av, _ := attributevalue.MarshalMap(s)

	// create input
	input := &dynamodb.PutItemInput{
		Item:         av,
		ReturnValues: dbTypes.ReturnValueNone,
		TableName:    &DynamoTable,
	}

	// add to table
	_, err := clients.DynamoClient.PutItem(context.TODO(), input)

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
	pkCondition := expression.Key(PartitionKey).Equal(expression.Value(s.PKVal()))
	skCondition := expression.Key(SortKey).Equal(expression.Value(s.SKVal()))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("SongID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for song Exists func")
	}

	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	// query
	result, err := clients.DynamoClient.Query(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("Error checking if song exists in table")
		return false, err
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
func (s *Song) Played(currentPlayCount int) error {
	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#PA": "PlayedAt",
			"#PP": "PlayedPosition",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":pk": &dbTypes.AttributeValueMemberS{Value: s.PKVal()},
			":sk": &dbTypes.AttributeValueMemberS{Value: s.SKVal()},
			":pa": &dbTypes.AttributeValueMemberS{Value: *s.PlayedAt},
			":pp": &dbTypes.AttributeValueMemberN{Value: strconv.Itoa(currentPlayCount)},
			":n":  &dbTypes.AttributeValueMemberS{Value: "NULL"},
		},
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: s.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: s.SKVal()},
		},
		ReturnValues:        dbTypes.ReturnValueNone,
		TableName:           &DynamoTable,
		ConditionExpression: aws.String("PK = :pk and SK = :sk and attribute_type(#PP, :n)"),
		UpdateExpression:    aws.String("SET #PA = :pa, #PP = :pp"),
	}

	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)

	if err != nil {
		// in the lead up to the day songs will get played twice - we don't want to mark them as played twice, only the first time
		// the condition will fail if the PlayedPosition attribute is not NULL, see line 171
		var crf *dbTypes.ConditionalCheckFailedException
		if errors.As(err, &crf) {
			logger.Log.Info().Str("songID", s.SongID).Msg("The song wasn't updated because it has already been played.")
			return nil
		}

		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("Error updating song")
		return err
	}

	// add it to the playlist and then increment the played count
	s.addToPlayedList()
	incrementPlayCount()
	return nil
}

// Get the song from the table
func (s *Song) Get() error {
	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: s.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: s.SKVal()},
		},
		TableName: &DynamoTable,
	}

	// getItem
	result, err := clients.DynamoClient.GetItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("error getting song from table")
		return err
	}

	if len(result.Item) == 0 {
		return errors.New("unable to find song in table")
	}

	// unmarshal item into struct
	err = attributevalue.UnmarshalMap(result.Item, &s)
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("failed to unmarshal dynamo item to group")
	}
	return nil
}

// incrementPlayCount increments the current playCount value
func incrementPlayCount() {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#V": "value",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":inc": &dbTypes.AttributeValueMemberN{Value: "1"},
		},
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: PlayCountPartitionKey},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: PlayCountSortKey},
		},
		ReturnValues:     dbTypes.ReturnValueNone,
		TableName:        &DynamoTable,
		UpdateExpression: aws.String("ADD #V :inc"),
	}
	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to increment the latest song position")
	}
}

// addToPlayedList adds the song to the list of played songs
func (s *Song) addToPlayedList() {
	av := &dbTypes.AttributeValueMemberS{Value: s.SongID}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#S": "SongIDs",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":s": &dbTypes.AttributeValueMemberL{Value: []dbTypes.AttributeValue{av}},
		},
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: PlayedSongsPartitionKey},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: PlayedSongsSortKey},
		},
		ReturnValues:     dbTypes.ReturnValueNone,
		TableName:        &DynamoTable,
		UpdateExpression: aws.String("SET #S = list_append(#S, :s)"),
	}
	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to add songID to played list")
	}
}
