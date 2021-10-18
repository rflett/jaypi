package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

type responseBody struct {
	Votes []types.Song `json:"votes"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get userId from pathParameters
	userID := request.PathParameters["userId"]

	// get user
	user := types.User{UserID: userID}

	// users can get themselves without doing the group check
	if authContext.UserID != userID {
		inSameGroup, err := services.UsersAreInSameGroup(authContext.UserID, userID)
		if err != nil {
			return services.ReturnError(err, http.StatusBadRequest)
		}
		if !inSameGroup {
			return services.ReturnError(errors.New("You have to a member of the group to do this"), http.StatusForbidden)
		}
	}

	// get their votes
	votes, voteErr := user.GetVotes()
	if voteErr == nil {
		user.Votes = &votes
	}

	// response
	rb := responseBody{Votes: votes}
	return services.ReturnJSON(rb, http.StatusOK)
}

func GetVotes(u string) ([]types.Song, error) {
	// get the users votes
	pkCondition := expression.Key(types.PartitionKey).Equal(expression.Value(fmt.Sprintf("%s#%s", types.UserPartitionKey, u)))
	skCondition := expression.Key(types.SortKey).BeginsWith(fmt.Sprintf("%s#", types.SongPartitionKey))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("rank"), expression.Name("songID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u).Msg("error building expression")
	}

	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: expr.Values(),
		ExpressionAttributeNames:  expr.Names(),
		KeyConditionExpression:    expr.KeyCondition(),
		ProjectionExpression:      expr.Projection(),
		FilterExpression:          expr.Filter(),
		TableName:                 &types.DynamoTable,
	}

	userVotes, err := clients.DynamoClient.Query(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u).Msg("error getting users votes")
		return []types.Song{}, err
	}

	var votes []types.Song = nil
	for _, vote := range userVotes.Items {
		song := types.Song{}
		var voteRank int
		if err = attributevalue.UnmarshalMap(vote, &song); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to unmarshal vote to songVote")
			continue
		}
		voteRank = *song.Rank
		logger.Log.Info().Msg(fmt.Sprintf("Saved rank as %d", voteRank))

		// fill out the rest of the song attributes
		if err = song.Get(); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to get song")
			continue
		}
		song.Rank = &voteRank
		logger.Log.Info().Msg(fmt.Sprintf("Returning song rank as %v", song))
		votes = append(votes, song)
	}
	return votes, nil
}

func main() {
	//lambda.Start(Handler)
	songs, _ := GetVotes("2e26e7dc-3f8c-456d-9d1b-8ce5b6447585")
	logger.Log.Info().Msg(fmt.Sprintf("%v", songs))
}
