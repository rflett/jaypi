package types

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
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
	Votes          *[]Song  `json:"votes" dynamodbav:"votes,omitemptyelem"`
	UpdatedAt      *string  `json:"updatedAt"`
	Password       *string  `json:"-" dynamodbav:"password"`
}

// UserClaims are the custom claims that embedded into the JWT token for authentication
type UserClaims struct {
	Name           string  `json:"name"`
	Picture        *string `json:"picture"`
	AuthProvider   string  `json:"https://delegator.com.au/AuthProvider"`
	AuthProviderId string  `json:"https://delegator.com.au/AuthProviderId"`
	jwt.StandardClaims
}

// userAuthProvider represents a user and their AuthProviderId
type userAuthProvider struct {
	PK             string `json:"-" dynamodbav:"PK"`
	SK             string `json:"-" dynamodbav:"SK"`
	UserID         string `json:"userID"`
	AuthProviderId string `json:"authProviderId"`
	AuthProvider   string `json:"authProvider"`
}

// songVote is a votes in a users top 10
type songVote struct {
	PK     string `json:"-" dynamodbav:"PK"`
	SK     string `json:"-" dynamodbav:"SK"`
	SongID string `json:"songID"`
	UserID string `json:"userID"`
	Rank   int    `json:"rank"`
}

// voteCount returns the number of votes a user already has
func (u *User) voteCount() (count int, error error) {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
			":sk": {
				S: aws.String(fmt.Sprintf("%s#", SongPrimaryKey)),
			},
		},
		KeyConditionExpression: aws.String("PK = :pk and begins_with(SK, :sk)"),
		ProjectionExpression:   aws.String("userID"),
		TableName:              &clients.DynamoTable,
	}

	queryResult, queryErr := clients.DynamoClient.Query(input)
	if queryErr != nil {
		logger.Log.Error().Err(queryErr).Str("userId", u.UserID).Msg("error getting user song count")
		return 0, queryErr
	}
	return int(*queryResult.Count), nil
}

// GenerateAvatarUrl generates a new avatar UUID and sets it on the user
func (u *User) GenerateAvatarUrl() (avatarUuid string, error error) {
	avatarUuid = uuid.NewString()
	avatarUrl := fmt.Sprintf("https://%s/user/avatar/%s.jpg", UserAvatarDomain, avatarUuid)

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#A": aws.String("avatarUrl"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":a": {
				S: &avatarUrl,
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserSortKey, u.UserID)),
			},
		},
		ReturnValues:     aws.String("NONE"),
		TableName:        &clients.DynamoTable,
		UpdateExpression: aws.String("SET #A = :a"),
	}

	_, err := clients.DynamoClient.UpdateItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error creating new avatar url for user")
		return "", err
	}

	return avatarUuid, nil
}

// Create the user and save them to the database
func (u *User) Create() (status int, error error) {
	// set fields
	u.UserID = uuid.NewString()
	u.PK = fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)
	u.SK = fmt.Sprintf("%s#%s", UserSortKey, u.UserID)
	u.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	// create item
	av, _ := dynamodbattribute.MarshalMap(u)

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
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("Error adding user to table")
		return http.StatusInternalServerError, err
	}
	logger.Log.Info().Str("userID", u.UserID).Msg("Successfully added user to table")

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
		ExpressionAttributeNames: map[string]*string{
			"#NN": aws.String("nickName"),
			"#UA": aws.String("updatedAt"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":nn": {
				S: aws.String(*u.NickName),
			},
			":ua": {
				S: u.UpdatedAt,
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserSortKey, u.UserID)),
			},
		},
		ReturnValues:     aws.String("NONE"),
		TableName:        &clients.DynamoTable,
		UpdateExpression: aws.String("SET #NN = :nn, #UA = :ua"),
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
			logger.Log.Error().Err(aerr).Str("userID", u.UserID).Msg("error updating user")
			return responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error updating user")
			return http.StatusInternalServerError, err
		}
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

	// set fields
	sv := songVote{
		PK:     fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID),
		SK:     fmt.Sprintf("%s#%s", SongPrimaryKey, s.SongID),
		SongID: s.SongID,
		UserID: u.UserID,
		Rank:   *s.Rank,
	}

	// create item
	av, _ := dynamodbattribute.MarshalMap(sv)
	input := &dynamodb.PutItemInput{
		Item:         av,
		ReturnValues: aws.String("NONE"),
		TableName:    &clients.DynamoTable,
	}

	// add to table
	_, err := clients.DynamoClient.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Str("songID", s.SongID).Msg("Error adding votes")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("userID", u.UserID).Str("songID", s.SongID).Msg("Added user votes")
	return http.StatusNoContent, nil
}

// RemoveVote removes a song as a users vote
func (u *User) RemoveVote(songID *string) (status int, error error) {
	// delete query
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", SongPrimaryKey, *songID)),
			},
		},
		TableName: &clients.DynamoTable,
	}

	// delete from table
	_, err := clients.DynamoClient.DeleteItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", *songID).Str("userID", u.UserID).Msg("Error removing song from user")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("songID", *songID).Str("userID", u.UserID).Msg("Removed song vote from user")
	return http.StatusNoContent, nil
}

// GetVotes returns a users votes
func (u *User) GetVotes() ([]Song, error) {
	// get the users votes
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
			":sk": {
				S: aws.String(fmt.Sprintf("%s#", SongPrimaryKey)),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#R": aws.String("rank"),
			"#S": aws.String("songID"),
		},
		KeyConditionExpression: aws.String("PK = :pk and begins_with(SK, :sk)"),
		ProjectionExpression:   aws.String("#S, #R"),
		TableName:              &clients.DynamoTable,
	}

	userVotes, err := clients.DynamoClient.Query(input)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error getting users votes")
		return []Song{}, err
	}

	var votes []Song = nil
	for _, vote := range userVotes.Items {
		song := Song{}
		var voteRank int
		if err = dynamodbattribute.UnmarshalMap(vote, &song); err != nil {
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
		logger.Log.Info().Msg(fmt.Sprintf("Returning song rank as %d", *song.Rank))
		votes = append(votes, song)
	}
	return votes, nil
}

// GetGroups returns the groups a user is a member of
func (u *User) GetGroups() ([]Group, error) {
	// get the users in the group
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("%s#", GroupPrimaryKey)),
			},
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
		},
		IndexName:              aws.String(GSI),
		ProjectionExpression:   aws.String("groupID"),
		KeyConditionExpression: aws.String("SK = :sk and begins_with(PK, :pk)"),
		TableName:              &clients.DynamoTable,
	}

	groupMemberships, err := clients.DynamoClient.Query(input)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error getting users groups")
		return []Group{}, err
	}

	var groups []Group = nil
	for _, membership := range groupMemberships.Items {
		group := Group{}
		if err = dynamodbattribute.UnmarshalMap(membership, &group); err != nil {
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
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserSortKey, u.UserID)),
			},
		},
		TableName: &clients.DynamoTable,
	}

	// getItem
	result, err := clients.DynamoClient.GetItem(input)

	// handle errors
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			var responseStatus int
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				responseStatus = http.StatusTooManyRequests
			case dynamodb.ErrCodeResourceNotFoundException:
				responseStatus = http.StatusNotFound
			case dynamodb.ErrCodeRequestLimitExceeded:
				responseStatus = http.StatusTooManyRequests
			case dynamodb.ErrCodeInternalServerError:
				responseStatus = http.StatusInternalServerError
			default:
				responseStatus = http.StatusInternalServerError
			}
			logger.Log.Error().Err(aerr).Str("userID", u.UserID).Msg("error getting user from table")
			return responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error getting user from table")
			return http.StatusInternalServerError, err
		}
	}

	if len(result.Item) == 0 {
		return http.StatusNotFound, errors.New("user not found")
	}

	// unmarshal item into struct
	err = dynamodbattribute.UnmarshalMap(result.Item, &u)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("failed to unmarshal dynamo item to user")
	}

	return http.StatusOK, nil
}

// GetByUserID the user from the table by their oauth id
func (u *User) GetByAuthProviderId() (status int, error error) {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s#%s", UserAuthProviderSortKey, *u.AuthProvider, *u.AuthProviderId)),
			},
			":pk": {
				S: aws.String(fmt.Sprintf("%s#", UserAuthProviderPrimaryKey)),
			},
		},
		IndexName:              aws.String(GSI),
		KeyConditionExpression: aws.String("SK = :sk and begins_with(PK, :pk)"),
		ProjectionExpression:   aws.String("userID"),
		TableName:              &clients.DynamoTable,
	}

	// query
	result, err := clients.DynamoClient.Query(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("providerID", *u.AuthProviderId).Msg("error querying by auth provider ID")
		return http.StatusInternalServerError, err
	}

	if len(result.Items) == 0 {
		return http.StatusNotFound, nil
	}

	// unmarshal item into user
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &u)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("failed to unmarshal dynamo item to user")
	}

	// fill the user out
	getUserStatus, _ := u.GetByUserID()
	return getUserStatus, nil
}

// Exists checks to see if a user exists. You can lookup via UserID or AuthProviderId.
func (u *User) Exists(lookup string) (bool, error) {
	var pk, sk *dynamodb.AttributeValue
	var kce string
	var idx *string

	switch lookup {
	case "UserID":
		pk = &dynamodb.AttributeValue{
			S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
		}
		sk = &dynamodb.AttributeValue{
			S: aws.String(fmt.Sprintf("%s#%s", UserSortKey, u.UserID)),
		}
		kce = "PK = :pk and SK = :sk"
		idx = nil
	case "AuthProviderId":
		pk = &dynamodb.AttributeValue{
			S: aws.String(fmt.Sprintf("%s#", UserAuthProviderPrimaryKey)),
		}
		sk = &dynamodb.AttributeValue{
			S: aws.String(fmt.Sprintf("%s#%s#%s", UserAuthProviderSortKey, *u.AuthProvider, *u.AuthProviderId)),
		}
		idx = aws.String(GSI)
		kce = "SK = :sk and begins_with(PK, :pk)"
	default:
		return false, errors.New("unsupported lookup, must be one of UserID, AuthProviderId")
	}

	// create input
	input := &dynamodb.QueryInput{
		ProjectionExpression:      aws.String("userID"),
		TableName:                 &clients.DynamoTable,
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":pk": pk, ":sk": sk},
		KeyConditionExpression:    aws.String(kce),
		IndexName:                 idx,
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
			logger.Log.Error().Err(err).Str("lookup", lookup).Msg("Error checking if user exists in table")
			return false, err
		} else {
			logger.Log.Error().Err(err).Str("lookup", lookup).Msg("Error checking if user exists in table")
			return false, err
		}
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
		PK:             fmt.Sprintf("%s#%s", UserAuthProviderPrimaryKey, u.UserID),
		SK:             fmt.Sprintf("%s#%s#%s", UserAuthProviderSortKey, *u.AuthProvider, *u.AuthProviderId),
		UserID:         u.UserID,
		AuthProviderId: *u.AuthProviderId,
		AuthProvider:   *u.AuthProvider,
	}

	// add the user auth provider to the table
	av, _ := dynamodbattribute.MarshalMap(uap)
	input := &dynamodb.PutItemInput{
		TableName:    &clients.DynamoTable,
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}
	_, err := clients.DynamoClient.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("Error adding user auth provider to table")
		return err
	}

	// success
	logger.Log.Info().Str("userID", u.UserID).Str("provider", *u.AuthProvider).Msg("Successfully user auth provider to table")
	return nil
}

// UpdatePoints adds the points to the users score
func (u *User) UpdatePoints(points int) error {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#P": aws.String("points"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":p": {
				N: aws.String(strconv.Itoa(points)),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserSortKey, u.UserID)),
			},
		},
		ReturnValues:     aws.String("NONE"),
		TableName:        &clients.DynamoTable,
		UpdateExpression: aws.String("ADD #P :p"),
	}
	_, err := clients.DynamoClient.UpdateItem(input)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("Unable to update the users points")
	}
	return nil
}

// LeaveGroup removes the User from a Group
func (u *User) LeaveGroup(groupID string) (status int, error error) {
	// delete query
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, groupID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
		},
		TableName: &clients.DynamoTable,
	}

	// delete membership from table
	_, err := clients.DynamoClient.DeleteItem(input)

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
	input := &secretsmanager.GetSecretValueInput{SecretId: &clients.JWTSigningSecret}
	secret, err := clients.SecretsClient.GetSecretValue(input)
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
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
			":sk": {
				S: aws.String(fmt.Sprintf("%s#", EndpointSortKey)),
			},
		},
		KeyConditionExpression: aws.String("PK = :pk and begins_with(SK, :sk)"),
		ProjectionExpression:   aws.String("arn, platform"),
		TableName:              &clients.DynamoTable,
	}

	queryResult, queryErr := clients.DynamoClient.Query(input)
	if queryErr != nil {
		logger.Log.Error().Err(queryErr).Str("userId", u.UserID).Msg("error getting users endpoints")
		return &[]PlatformEndpoint{}, queryErr
	}

	var endpoints []PlatformEndpoint
	for _, item := range queryResult.Items {
		endpoint := PlatformEndpoint{}
		if err := dynamodbattribute.UnmarshalMap(item, &endpoint); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to unmarshal item to PlatformEndpoint")
			continue
		}
		endpoints = append(endpoints, endpoint)
	}

	return &endpoints, nil
}
