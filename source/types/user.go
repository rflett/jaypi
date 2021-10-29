package types

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/dgrijalva/jwt-go"
	sentryGo "github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"net/http"
	"strconv"
	"time"
)

// User is a User of the application
type User struct {
	PK             string   `json:"-" dynamodbav:"PK"`
	SK             string   `json:"-" dynamodbav:"SK"`
	UserID         string   `json:"userID"`
	Name           string   `json:"name"`
	Email          string   `json:"email"`
	Points         int      `json:"points"`
	CreatedAt      string   `json:"createdAt"`
	Groups         *[]Group `json:"groups" dynamodbav:"-"`
	NickName       *string  `json:"nickName"`
	AuthProvider   *string  `json:"authProvider"`
	AuthProviderId *string  `json:"authProviderId"`
	AvatarUrl      *string  `json:"avatarUrl"`
	Votes          *[]Song  `json:"votes" dynamodbav:"Votes,omitemptyelem"`
	UpdatedAt      *string  `json:"updatedAt"`
	Password       *string  `json:"-"`
}

// return the partition key value for a user
func (u *User) PKVal() string {
	return fmt.Sprintf("%s#%s", UserPartitionKey, u.UserID)
}

// return the sort key value for a user
func (u *User) SKVal() string {
	return fmt.Sprintf("%s#%s", UserSortKey, u.UserID)
}

// voteCount returns the number of votes a user already has
func (u *User) voteCount() (count int, error error) {
	pkCondition := expression.Key(PartitionKey).Equal(expression.Value(u.PKVal()))
	skCondition := expression.Key(SortKey).BeginsWith(fmt.Sprintf("%s#", SongPartitionKey))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("SongID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for voteCount func")
	}

	// input
	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	queryResult, queryErr := clients.DynamoClient.Query(context.TODO(), input)
	if queryErr != nil {
		logger.Log.Error().Err(queryErr).Str("userId", u.UserID).Msg("error querying user voteCount")
		return 0, queryErr
	}

	return int(queryResult.Count), nil
}

// GenerateAvatarUrl generates a new avatar UUID and sets it on the user
func (u *User) GenerateAvatarUrl() (avatarUuid string, error error) {
	avatarUuid = uuid.NewString()
	avatarUrl := fmt.Sprintf("https://%s/user/avatar/%s.jpg", UserAvatarDomain, avatarUuid)

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#A": "avatarUrl",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":a": &dbTypes.AttributeValueMemberS{Value: avatarUrl},
		},
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: u.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: u.SKVal()},
		},
		ReturnValues:     dbTypes.ReturnValueNone,
		TableName:        &DynamoTable,
		UpdateExpression: aws.String("SET #A = :a"),
	}

	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error updating user avatarUrl")
		return "", err
	}

	return avatarUuid, nil
}

// Create the user and save them to the database
func (u *User) Create() (status int, error error) {
	// set fields
	u.UserID = uuid.NewString()
	u.PK = u.PKVal()
	u.SK = u.SKVal()
	u.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// create item
	av, _ := attributevalue.MarshalMap(u)

	// create input
	input := &dynamodb.PutItemInput{
		TableName:    &DynamoTable,
		Item:         av,
		ReturnValues: dbTypes.ReturnValueNone,
	}

	// add to table
	_, err := clients.DynamoClient.PutItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error putting user in db")
		return http.StatusInternalServerError, err
	}
	logger.Log.Info().Str("userID", u.UserID).Msg("Successfully put user in db")

	// create their auth provider
	authProviderErr := u.NewAuthProvider()
	if authProviderErr != nil {
		return http.StatusInternalServerError, authProviderErr
	}

	// ok!
	sentryGo.ConfigureScope(func(scope *sentryGo.Scope) {
		scope.SetUser(sentryGo.User{
			ID:        u.UserID,
			Username:  *u.AuthProviderId,
			IPAddress: "{{auto}}",
		})
		scope.SetTag("AuthProvider", *u.AuthProvider)
	})
	sentryGo.CaptureMessage("User signed up!")
	return http.StatusCreated, nil
}

// Update the user's attributes
func (u *User) Update() (status int, error error) {
	// set fields
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	u.UpdatedAt = &updatedAt

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#NN": "nickName",
			"#UA": "updatedAt",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":nn": &dbTypes.AttributeValueMemberS{Value: *u.NickName},
			":ua": &dbTypes.AttributeValueMemberS{Value: *u.UpdatedAt},
		},
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: u.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: u.SKVal()},
		},
		ReturnValues:     dbTypes.ReturnValueNone,
		TableName:        &DynamoTable,
		UpdateExpression: aws.String("SET #NN = :nn, #UA = :ua"),
	}

	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error updating user item")
		return http.StatusInternalServerError, err
	}

	return http.StatusNoContent, nil
}

// AddVote adds a song as a votes for the user
func (u *User) AddVote(s *Song) (status int, error error) {
	// check if song exists and add it if it doesn't
	exists, existsErr := s.Exists()
	if existsErr != nil {
		return http.StatusInternalServerError, existsErr
	}

	if !exists {
		createSongErr := s.Create()
		if createSongErr != nil {
			return http.StatusInternalServerError, createSongErr
		}
	}

	// don't allow more than 10 votes
	vc, vcErr := u.voteCount()
	if vcErr != nil {
		return http.StatusInternalServerError, vcErr
	}
	if vc >= 10 {
		tooManyCountsErr := errors.New("user already has 10 song votes")
		logger.Log.Error().Err(tooManyCountsErr).Str("userID", u.UserID).Msg("User has maxed out their votes")
		return http.StatusBadRequest, tooManyCountsErr
	}

	// create item
	av, _ := attributevalue.MarshalMap(songVote{
		PK:     u.PKVal(),
		SK:     s.PKVal(),
		SongID: s.SongID,
		UserID: u.UserID,
		Rank:   *s.Rank,
	})

	input := &dynamodb.PutItemInput{
		Item:         av,
		ReturnValues: dbTypes.ReturnValueNone,
		TableName:    &DynamoTable,
	}

	// add to table
	_, err := clients.DynamoClient.PutItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Str("songID", s.SongID).Msg("Error adding vote for user")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("userID", u.UserID).Str("songID", s.SongID).Int("Rank", *s.Rank).Msg("Added user vote")
	return http.StatusNoContent, nil
}

// RemoveVote removes a song as a users vote
func (u *User) RemoveVote(songID *string) (status int, error error) {
	// delete query
	input := &dynamodb.DeleteItemInput{
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{
				Value: u.PKVal(),
			},
			SortKey: &dbTypes.AttributeValueMemberS{
				Value: fmt.Sprintf("%s#%s", SongPartitionKey, *songID),
			},
		},
		TableName: &DynamoTable,
	}

	// delete from table
	_, err := clients.DynamoClient.DeleteItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", *songID).Str("userID", u.UserID).Msg("Error removing users vote")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("songID", *songID).Str("userID", u.UserID).Msg("Removed song vote from user")
	return http.StatusNoContent, nil
}

// GetVotes returns a users votes
func (u *User) GetVotes() ([]Song, error) {
	// get the users votes
	pkCondition := expression.Key(PartitionKey).Equal(expression.Value(u.PKVal()))
	skCondition := expression.Key(SortKey).BeginsWith(fmt.Sprintf("%s#", SongPartitionKey))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()

	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error building GetVotes expression")
	}

	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	userVotes, err := clients.DynamoClient.Query(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error getting users votes")
		return []Song{}, err
	}

	var votes []Song = nil
	for _, vote := range userVotes.Items {

		// unmarshall the vote
		songVote := songVote{}
		if err = attributevalue.UnmarshalMap(vote, &songVote); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to unmarshal vote to songVote")
			continue
		}

		// convert the vote to a full song record
		song := Song{}
		song, err = songVote.GetAsSong()
		if err != nil {
			logger.Log.Error().Err(err).Msg("Unable to get songVote as song")
			continue
		}

		// append to return object
		votes = append(votes, song)
	}
	return votes, nil
}

// GetGroups returns the groups a user is a member of
func (u *User) GetGroups() ([]Group, error) {
	// get the users in the group
	pkCondition := expression.Key(PartitionKey).BeginsWith(fmt.Sprintf("%s#", GroupPartitionKey))
	skCondition := expression.Key(SortKey).Equal(expression.Value(u.PKVal()))
	keyCondition := expression.KeyAnd(skCondition, pkCondition)

	projExpr := expression.NamesList(expression.Name("GroupID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error building GetGroups expression")
	}

	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		IndexName:                 aws.String(GSI),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	groupMemberships, err := clients.DynamoClient.Query(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error querying users groups")
		return []Group{}, err
	}

	var groups []Group = nil
	for _, membership := range groupMemberships.Items {
		group := Group{}
		if err = attributevalue.UnmarshalMap(membership, &group); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to unmarshal group to Group")
			continue
		}
		_, _ = group.Get()
		groups = append(groups, group)
	}
	return groups, nil
}

// GetByUserID the user from the table
func (u *User) GetByUserID() (status int, error error) {
	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: u.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: u.SKVal()},
		},
		TableName: &DynamoTable,
	}

	// getItem
	result, err := clients.DynamoClient.GetItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error getting user from database")
		return http.StatusInternalServerError, err
	}

	if len(result.Item) == 0 {
		return http.StatusNotFound, errors.New("user not found")
	}

	// unmarshal item into struct
	err = attributevalue.UnmarshalMap(result.Item, &u)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("failed to unmarshal dynamo item map to user")
	}

	return http.StatusOK, nil
}

// GetByUserID the user from the table by their oauth id
func (u *User) GetByAuthProviderId() (status int, error error) {
	// input
	pkCondition := expression.Key(PartitionKey).BeginsWith(fmt.Sprintf("%s#", UserAuthProviderPartitionKey))
	skCondition := expression.Key(SortKey).Equal(
		expression.Value(fmt.Sprintf("%s#%s#%s", UserAuthProviderSortKey, *u.AuthProvider, *u.AuthProviderId)),
	)
	keyCondition := expression.KeyAnd(skCondition, pkCondition)

	projExpr := expression.NamesList(expression.Name("UserID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error building GetByAuthProviderId expression")
	}

	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		IndexName:                 aws.String(GSI),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	// query
	result, err := clients.DynamoClient.Query(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("providerID", *u.AuthProviderId).Msg("error querying by auth provider ID")
		return http.StatusInternalServerError, err
	}

	if len(result.Items) == 0 {
		return http.StatusNotFound, nil
	}

	// unmarshal item into user
	err = attributevalue.UnmarshalMap(result.Items[0], &u)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("failed to unmarshal dynamo item map to user")
	}

	// fill the user out
	getUserStatus, _ := u.GetByUserID()
	return getUserStatus, nil
}

// Exists checks to see if a user exists. You can lookup via UserID or AuthProviderId.
func (u *User) Exists(lookup string) (bool, error) {
	var pkCondition, skCondition expression.KeyConditionBuilder
	var idx *string

	switch lookup {
	case "UserID":
		pkCondition = expression.Key(PartitionKey).Equal(expression.Value(u.PKVal()))
		skCondition = expression.Key(SortKey).Equal(expression.Value(u.SKVal()))
	case "AuthProviderId":
		idx = aws.String(GSI)
		pkCondition = expression.Key(PartitionKey).BeginsWith(fmt.Sprintf("%s#", UserAuthProviderPartitionKey))
		skCondition = expression.Key(SortKey).Equal(
			expression.Value(
				fmt.Sprintf("%s#%s#%s", UserAuthProviderSortKey, *u.AuthProvider, *u.AuthProviderId),
			),
		)
	default:
		return false, errors.New("unsupported lookup, must be one of UserID, AuthProviderId")
	}

	projExpr := expression.NamesList(expression.Name("UserID"))
	keyCondition := expression.KeyAnd(skCondition, pkCondition)
	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error building user Exists expression")
	}

	// create input
	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		IndexName:                 idx,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	// query
	result, err := clients.DynamoClient.Query(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("lookup", lookup).Msg("Error checking if user exists in table")
		return false, err
	}

	// user doesn't exist
	if len(result.Items) == 0 {
		logger.Log.Info().Str("lookup", lookup).Msg("User does not exist in table")
		return false, nil
	}

	// user exists
	logger.Log.Info().Str("lookup", lookup).Msg("User already exists in table")
	return true, nil
}

// NewAuthProvider creates a new mapping of a user to their auth provider
func (u *User) NewAuthProvider() error {
	uap := userAuthProvider{
		PK:             fmt.Sprintf("%s#%s", UserAuthProviderPartitionKey, u.UserID),
		SK:             fmt.Sprintf("%s#%s#%s", UserAuthProviderSortKey, *u.AuthProvider, *u.AuthProviderId),
		UserID:         u.UserID,
		AuthProviderId: *u.AuthProviderId,
		AuthProvider:   *u.AuthProvider,
	}

	// add the user auth provider to the table
	av, _ := attributevalue.MarshalMap(uap)
	input := &dynamodb.PutItemInput{
		TableName:    &DynamoTable,
		Item:         av,
		ReturnValues: dbTypes.ReturnValueNone,
	}
	_, err := clients.DynamoClient.PutItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Str("provider", *u.AuthProvider).Msg("Error putting NewAuthProvider for user in database")
		return err
	}

	// success
	logger.Log.Info().Str("userID", u.UserID).Str("provider", *u.AuthProvider).Msg("Successfully put NewAuthProvider in database")
	return nil
}

// UpdatePoints adds the points to the users score
func (u *User) UpdatePoints(points int) error {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#P": "points",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":p": &dbTypes.AttributeValueMemberN{Value: strconv.Itoa(points)},
		},
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: u.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: u.SKVal()},
		},
		ReturnValues:     dbTypes.ReturnValueNone,
		TableName:        &DynamoTable,
		UpdateExpression: aws.String("ADD #P :p"),
	}
	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("Unable to update the users points")
	}
	return nil
}

// LeaveGroup removes the User from a Group
func (u *User) LeaveGroup(groupID string) (status int, error error) {
	// delete query
	input := &dynamodb.DeleteItemInput{
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: fmt.Sprintf("%s#%s", GroupPartitionKey, groupID)},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: u.PKVal()},
		},
		TableName: &DynamoTable,
	}

	// delete membership from table
	_, err := clients.DynamoClient.DeleteItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", groupID).Str("userID", u.UserID).Msg("Error removing user from group")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("userID", u.UserID).Str("groupID", groupID).Msg("User left group")
	return http.StatusNoContent, nil
}

// CreateToken returns a new CreateToken for the user
func (u *User) CreateToken() (string, error) {
	// create the token
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims = &UserClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "delegator.com.au",
			Subject:   u.UserID,
			Audience:  "delegator.com.au",
			ExpiresAt: time.Now().Add(time.Hour * 72).Unix(),
			NotBefore: time.Now().Unix(),
			IssuedAt:  time.Now().Unix(),
			Id:        uuid.NewString(),
		},
		Name:           u.Name,
		Picture:        u.AvatarUrl,
		AuthProvider:   *u.AuthProvider,
		AuthProviderId: *u.AuthProviderId,
	}

	// get the signing key
	input := &secretsmanager.GetSecretValueInput{SecretId: &JWTSigningSecret}
	secret, err := clients.SecretsClient.GetSecretValue(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Msg("unable to get signing key from secretsmanager")
		return "", err
	}

	// parse the key, sign the token and return it
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(*secret.SecretString))
	if err != nil {
		logger.Log.Error().Err(err).Msg("unable to parse signing private key")
		return "", err
	}
	return token.SignedString(key)
}

// GetEndpoints returns all of the device endpoints that a user has
func (u *User) GetEndpoints() (*[]PlatformEndpoint, error) {
	// input
	pkCondition := expression.Key(PartitionKey).Equal(expression.Value(u.PKVal()))
	skCondition := expression.Key(SortKey).BeginsWith(fmt.Sprintf("%s#", EndpointSortKey))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("Arn"), expression.Name("Platform"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for GetEndpoints func")
	}

	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	queryResult, queryErr := clients.DynamoClient.Query(context.TODO(), input)
	if queryErr != nil {
		logger.Log.Error().Err(queryErr).Str("userId", u.UserID).Msg("error getting users endpoints")
		return &[]PlatformEndpoint{}, queryErr
	}

	var endpoints []PlatformEndpoint
	for _, item := range queryResult.Items {
		endpoint := PlatformEndpoint{}
		if err := attributevalue.UnmarshalMap(item, &endpoint); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to unmarshal item to PlatformEndpoint")
			continue
		}
		endpoints = append(endpoints, endpoint)
	}

	return &endpoints, nil
}
