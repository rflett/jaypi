package types

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"net/http"
	"time"
)

// A Game (usually drinking related) that people get to play when someone in their group has a song vote played
type Game struct {
	PK          string  `json:"-" dynamodbav:"PK"`
	SK          string  `json:"-" dynamodbav:"SK"`
	GameID      string  `json:"gameID"`
	GroupID     string  `json:"groupID"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   *string `json:"updatedAt"`
}

// Create the game and save it to the database
func (g *Game) Create() (status int, error error) {
	// set fields
	g.GameID = uuid.NewString()
	g.PK = fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)
	g.SK = fmt.Sprintf("%s#%s", GameSortKey, g.GameID)
	g.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// create item
	av, _ := dynamodbattribute.MarshalMap(g)

	// create input
	input := &dynamodb.PutItemInput{
		TableName:    &clients.DynamoTable,
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}

	// add to table
	_, err := clients.DynamoClient.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("Error adding game to table")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("Successfully added game to table")
	return http.StatusCreated, nil
}

// Update the game's attributes
func (g *Game) Update() (status int, error error) {
	// set fields
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	g.UpdatedAt = &updatedAt

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#N":  aws.String("name"),
			"#D":  aws.String("description"),
			"#UA": aws.String("updatedAt"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":ua": {
				S: g.UpdatedAt,
			},
			":n": {
				S: &g.Name,
			},
			":d": {
				S: &g.Description,
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", GameSortKey, g.GameID)),
			},
		},
		ReturnValues:     aws.String("NONE"),
		TableName:        &clients.DynamoTable,
		UpdateExpression: aws.String("SET #N = :n, #UA = :ua, #D = :d"),
	}

	_, err := clients.DynamoClient.UpdateItem(input)

	// handle errors
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			var responseStatus int
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				responseStatus = http.StatusTooManyRequests
			case dynamodb.ErrCodeResourceNotFoundException:
				responseStatus = http.StatusNotFound
			case dynamodb.ErrCodeConditionalCheckFailedException:
				responseStatus = http.StatusNotFound
			case dynamodb.ErrCodeRequestLimitExceeded:
				responseStatus = http.StatusTooManyRequests
			case dynamodb.ErrCodeInternalServerError:
				responseStatus = http.StatusInternalServerError
			default:
				responseStatus = http.StatusInternalServerError
			}
			logger.Log.Error().Err(aerr).Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("error updating game")
			return responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("error updating game")
			return http.StatusInternalServerError, err
		}
	}

	return http.StatusNoContent, nil
}

// RemoveVote removes a song as a users vote
func (g *Game) Delete() (status int, error error) {
	// delete query
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", GameSortKey, g.GameID)),
			},
		},
		TableName: &clients.DynamoTable,
	}

	// delete from table
	_, err := clients.DynamoClient.DeleteItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("error deleting game")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("succesfully deleted game")
	return http.StatusNoContent, nil
}
