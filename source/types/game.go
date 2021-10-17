package types

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
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

func (g *Game) PKVal() string {
	return fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)
}

func (g *Game) SKVal() string {
	return fmt.Sprintf("%s#%s", GameSortKey, g.GameID)
}

// Create the game and save it to the database
func (g *Game) Create() (status int, error error) {
	// set fields
	g.GameID = uuid.NewString()
	g.PK = g.PKVal()
	g.SK = g.SKVal()
	g.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// create item
	av, _ := attributevalue.MarshalMap(g)

	// create input
	input := &dynamodb.PutItemInput{
		TableName:    &clients.DynamoTable,
		Item:         av,
		ReturnValues: dbTypes.ReturnValueNone,
	}

	// add to table
	_, err := clients.DynamoClient.PutItem(context.TODO(), input)

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
		ExpressionAttributeNames: map[string]string{
			"#N":  "name",
			"#D":  "description",
			"#UA": "updatedAt",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":ua": &dbTypes.AttributeValueMemberS{Value: *g.UpdatedAt},
			":n":  &dbTypes.AttributeValueMemberS{Value: g.Name},
			":d":  &dbTypes.AttributeValueMemberS{Value: g.Description},
		},
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: g.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: g.SKVal()},
		},
		ReturnValues:     dbTypes.ReturnValueNone,
		TableName:        &clients.DynamoTable,
		UpdateExpression: aws.String("SET #N = :n, #UA = :ua, #D = :d"),
	}

	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("error updating game")
		return http.StatusInternalServerError, err
	}

	return http.StatusNoContent, nil
}

// Delete removes the game from the database
func (g *Game) Delete() (status int, error error) {
	// delete query
	input := &dynamodb.DeleteItemInput{
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: g.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: g.SKVal()},
		},
		TableName: &clients.DynamoTable,
	}

	// delete from table
	_, err := clients.DynamoClient.DeleteItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("error deleting game")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("succesfully deleted game")
	return http.StatusNoContent, nil
}

// Exists checks to see if the Game exists in the table already
func (g *Game) Exists() (bool, error) {
	// input
	pkCondition := expression.Key(PartitionKey).Equal(expression.Value(g.PKVal()))
	skCondition := expression.Key(SortKey).Equal(expression.Value(g.SKVal()))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("gameID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for voteCount func")
	}

	input := &dynamodb.QueryInput{
		TableName:                 &clients.DynamoTable,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	// query
	result, err := clients.DynamoClient.Query(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("Error checking if game exists in table")
		return false, err
	}

	// game doesn't exist
	if len(result.Items) == 0 {
		logger.Log.Info().Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("Game does not exist in table")
		return false, nil
	}

	// game exists
	logger.Log.Info().Str("groupID", g.GroupID).Str("gameID", g.GameID).Msg("Game already exists in table")
	return true, nil
}
