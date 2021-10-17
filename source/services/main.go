package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	sentryGo "github.com/getsentry/sentry-go"
	"golang.org/x/crypto/bcrypt"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"sort"
)

// GetRecentlyPlayed returns the songs that have been played
func GetRecentlyPlayed() ([]types.Song, error) {
	// input
	condition := expression.Name(types.PartitionKey).BeginsWith(fmt.Sprintf("%s#", types.SongPrimaryKey))
	expr, err := expression.NewBuilder().WithCondition(condition).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for GetRecentlyPlayed func")
	}

	input := &dynamodb.ScanInput{
		TableName:                 &clients.DynamoTable,
		Limit:                     aws.Int32(100),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	// get songs from db
	recentSongs, err := clients.DynamoClient.Scan(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Msg("error getting recent songs")
		return []types.Song{}, err
	}

	// unmarshal songs into slice
	var songs []types.Song = nil
	for _, recentSong := range recentSongs.Items {
		song := types.Song{}
		if err = attributevalue.UnmarshalMap(recentSong, &song); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to unmarshal recentSong to song")
			continue
		}
		songs = append(songs, song)
	}

	// sort the songs by PlayedPosition asc
	sort.Slice(songs, func(i, j int) bool {
		return *songs[i].PlayedPosition < *songs[j].PlayedPosition
	})

	// return
	return songs, nil
}

// GetGroupFromCode returns the groupID based on the group code
func GetGroupFromCode(code string) (*types.Group, error) {
	// input
	pkCondition := expression.Key(types.PartitionKey).BeginsWith(fmt.Sprintf("%s#", types.GroupCodePrimaryKey))
	skCondition := expression.Key(types.SortKey).Equal(expression.Value(fmt.Sprintf("%s#%s", types.GroupCodeSortKey, code)))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("groupID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for GetGroupFromCode func")
	}

	input := &dynamodb.QueryInput{
		TableName:                 &clients.DynamoTable,
		IndexName:                 aws.String(types.GSI),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	// query
	result, err := clients.DynamoClient.Query(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("code", code).Msg("error querying code")
		return &types.Group{}, err
	}

	// code doesn't exist
	if len(result.Items) == 0 {
		codeNotExistErr := errors.New("Group code not found.")
		logger.Log.Error().Err(codeNotExistErr).Str("code", code).Msg(" groupcode does not exist")
		return &types.Group{}, codeNotExistErr
	}

	// unmarshal groupID into the Group struct
	gc := types.GroupCode{}
	err = attributevalue.UnmarshalMap(result.Items[0], &gc)
	if err != nil {
		logger.Log.Error().Err(err).Str("code", code).Msg("error unmarshalling code to GroupCode")
		fmt.Printf("Failed to unmarshal Record, %v", err)
		return &types.Group{}, err
	}

	// get the group
	g := &types.Group{GroupID: gc.GroupID}
	_, getGroupErr := g.Get()
	if getGroupErr != nil {
		return g, err
	}
	return g, nil
}

// HashAndSaltPassword generates a salt and hashes a password with it using bcrypt
func HashAndSaltPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to generate hashed password")
		return "", err
	}
	return string(hashed), nil
}

// ComparePasswords compares a hashed password with a plain text one and sees if they match
func ComparePasswords(hashedPassword string, textPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(textPassword))
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to compare passwords")
		return false
	}
	return true
}

// GetOauthProvider retrieves an oauth provider by its string name
func GetOauthProvider(providerName string) (*types.OauthProvider, error) {
	provider, exists := types.OauthProviders[providerName]

	if !exists {
		logger.Log.Warn().Msg(fmt.Sprintf("Unhandled provider requested %s", providerName))
		return nil, fmt.Errorf("Sorry. That oauth provider isn't supported.")
	}

	return provider, nil
}

// GetAuthorizerContext returns the AuthorizerContext from the APIGatewayProxyRequestContext
func GetAuthorizerContext(ctx events.APIGatewayProxyRequestContext) *types.AuthorizerContext {
	var AuthProvider = ctx.Authorizer["AuthProvider"].(string)
	var AuthProviderId = ctx.Authorizer["AuthProviderId"].(string)
	var Name = ctx.Authorizer["Name"].(string)
	var UserID = ctx.Authorizer["UserID"].(string)

	sentryGo.ConfigureScope(func(scope *sentryGo.Scope) {
		scope.SetUser(sentryGo.User{
			ID:        UserID,
			Username:  AuthProviderId,
			IPAddress: "{{auto}}",
		})
		scope.SetTag("AuthProvider", AuthProvider)
	})

	return &types.AuthorizerContext{
		AuthProvider:   AuthProvider,
		AuthProviderId: AuthProviderId,
		Name:           Name,
		UserID:         UserID,
	}
}

// GetPlatformEndpointAttributes returns a map of the endpoints attributes
func GetPlatformEndpointAttributes(arn string) (map[string]*string, error) {
	input := &sns.GetEndpointAttributesInput{EndpointArn: &arn}
	attributes, err := clients.SNSClient.GetEndpointAttributes(input)
	if err != nil {
		logger.Log.Error().Err(err).Str("platformEndpointArn", arn).Msg("Error getting platform endpoint attributes")
	}
	return attributes.Attributes, err
}

// UserIsInGroup returns whether a user is a member of a group
func UserIsInGroup(userID string, groupID string) (bool, error) {
	// input
	pkCondition := expression.Key(types.PartitionKey).Equal(
		expression.Value(fmt.Sprintf("%s#%s", types.GroupPrimaryKey, groupID)),
	)
	skCondition := expression.Key(types.SortKey).Equal(
		expression.Value(fmt.Sprintf("%s#%s", types.UserPrimaryKey, userID)),
	)
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("userID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for UserIsInGroup func")
	}

	input := &dynamodb.QueryInput{
		TableName:                 &clients.DynamoTable,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		ProjectionExpression:      expr.Projection(),
	}

	// query
	result, err := clients.DynamoClient.Query(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", userID).Str("groupID", groupID).Msg("Unable to check if user is in group")
		return false, err
	}

	return len(result.Items) != 0, nil
}

// UsersAreInSameGroup returns whether two users are in the same group
func UsersAreInSameGroup(userIdA string, userIdB string) (bool, error) {
	userA := types.User{UserID: userIdA}
	userB := types.User{UserID: userIdB}

	userAGroups, err := userA.GetGroups()
	if err != nil {
		return false, err
	}

	userBGroups, err := userB.GetGroups()
	if err != nil {
		return false, err
	}

	inSameGroup := false
	for _, groupA := range userAGroups {
		for _, groupB := range userBGroups {
			if groupA.GroupID == groupB.GroupID {
				inSameGroup = true
				break
			}
		}
		if inSameGroup {
			break
		}
	}

	return inSameGroup, nil
}

// UserIsGroupOwner returns whether the user is the group owner
func UserIsGroupOwner(userID string, groupID string) (bool, error) {
	// get the group
	group := types.Group{GroupID: groupID}
	_, err := group.Get()

	if err != nil {
		logger.Log.Error().Err(err).Str("userID", userID).Str("groupID", groupID).Msg("Unable to check if user is the group owner")
		return false, err
	}

	return group.OwnerID == userID, nil
}

// ReturnNoContent returns an APIGW response with no content
func ReturnNoContent() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

// ReturnJSON returns a struct in an APIGW response body
func ReturnJSON(body interface{}, status int) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	marshalledBody, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{Body: string(marshalledBody), StatusCode: status, Headers: headers}, nil
}

// ReturnError returns an error from APIGW in a standard format
func ReturnError(err error, status int) (events.APIGatewayProxyResponse, error) {
	sentryGo.CaptureException(err)
	logger.Log.Info().Str("status", string(rune(status))).Str("err", err.Error()).Msg("Returning error from APIGW")
	body := map[string]interface{}{
		"success": false,
		"error":   err.Error(),
	}
	return ReturnJSON(body, status)
}

// PurgeSongs removes all songs from the table
func PurgeSongs() {
	// input
	input := &dynamodb.ScanInput{
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":pk": &dbTypes.AttributeValueMemberS{Value: fmt.Sprintf("%s#", types.SongPrimaryKey)},
		},
		FilterExpression: aws.String("begins_with(PK, :pk)"),
		TableName:        &clients.DynamoTable,
	}

	paginator := dynamodb.NewScanPaginator(clients.DynamoClient, input)

	for paginator.HasMorePages() {
		page, pageErr := paginator.NextPage(context.TODO())
		if pageErr != nil {
			logger.Log.Error().Err(pageErr).Msg("error getting NextPage from PurgeSongs paginator")
			break
		}

		for _, item := range page.Items {
			song := types.Song{}
			unMarshErr := attributevalue.UnmarshalMap(item, &song)
			if unMarshErr != nil {
				logger.Log.Error().Err(unMarshErr).Msg("error unmarshalling item to song")
			} else {
				_ = song.Delete()
			}
		}
	}
}

// SetPlayCount sets the current playCount to a specific value
func SetPlayCount(val string) {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#V": "value",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":val": &dbTypes.AttributeValueMemberS{Value: val},
		},
		Key: map[string]dbTypes.AttributeValue{
			types.PartitionKey: &dbTypes.AttributeValueMemberS{Value: types.PlayCountPrimaryKey},
			types.SortKey:      &dbTypes.AttributeValueMemberS{Value: types.PlayCountSortKey},
		},
		ReturnValues:     dbTypes.ReturnValueNone,
		TableName:        &clients.DynamoTable,
		UpdateExpression: aws.String("SET #V = :val"),
	}
	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to set the play count")
	}
}
