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
	"github.com/google/uuid"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"net/http"
	"strconv"
	"time"
)

// User is a User of the application
type User struct {
	PK             string  `json:"-" dynamodbav:"PK"`
	SK             string  `json:"-" dynamodbav:"SK"`
	UserID         string  `json:"userID"`
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	Points         int     `json:"points"`
	CreatedAt      string  `json:"createdAt"`
	NickName       *string `json:"nickName"`
	Password       *string `json:"-" dynamodbav:"password"`
	AuthProvider   *string `json:"authProvider"`
	AuthProviderId *string `json:"authProviderId"`
	AvatarUrl      *string `json:"avatarUrl"`
	UpdatedAt      *string `json:"updatedAt"`
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
	PK       string `json:"-" dynamodbav:"PK"`
	SK       string `json:"-" dynamodbav:"SK"`
	SongID   string `json:"songID"`
	UserID   string `json:"userID"`
	Position int    `json:"position"`
}

type userClaims struct {
	Name           string  `json:"name"`
	AuthProvider   string  `json:"https://delegator.com.au/AuthProvider"`
	AuthProviderId string  `json:"https://delegator.com.au/AuthProviderId"`
	Picture        *string `json:"picture"`
	jwt.StandardClaims
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
				S: aws.String(fmt.Sprintf("%s#", "SONG")),
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
	return http.StatusCreated, nil
}

// Update the user's attributes
func (u *User) Update() (status int, error error) {
	// set fields
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	u.UpdatedAt = &updatedAt

	pk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
	}
	sk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", UserSortKey, u.UserID)),
	}

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#NN": aws.String("nickName"),
			"#UA": aws.String("updatedAt"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": &pk,
			":sk": &sk,
			":nn": {
				S: aws.String(*u.NickName),
			},
			":ua": {
				S: aws.String(*u.UpdatedAt),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": &pk,
			"SK": &sk,
		},
		ReturnValues:        aws.String("NONE"),
		TableName:           &clients.DynamoTable,
		ConditionExpression: aws.String("PK = :pk and SK = :sk"),
		UpdateExpression:    aws.String("SET #NN = :nn, #UA = :ua"),
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
func (u *User) AddVote(s *Song, position int) (status int, error error) {
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
		logger.Log.Error().Err(tooManyCountsErr).Str("userID", u.UserID).Msg("User has maxxed out their votes")
		return http.StatusBadRequest, tooManyCountsErr
	}

	// set fields
	sv := songVote{
		PK:       fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID),
		SK:       fmt.Sprintf("%s#%s", "SONG", s.SongID),
		SongID:   s.SongID,
		UserID:   u.UserID,
		Position: position,
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
	// set fields
	pk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
	}
	sk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", "SONG", *songID)),
	}

	// delete query
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": &pk,
			"SK": &sk,
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
		return http.StatusNotFound, nil
	}

	// unmarshal item into struct
	err = dynamodbattribute.UnmarshalMap(result.Item, &u)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("failed to unmarshal dynamo item to user")
	}

	return http.StatusOK, nil
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

// LeaveAllGroups removes a member from all of their groups
func (u *User) LeaveAllGroups() error {
	// find any other groups to leave
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
			":pk": {
				S: aws.String(fmt.Sprintf("%s#", "GROUP")),
			},
		},
		IndexName:              aws.String(GSI),
		KeyConditionExpression: aws.String("SK = :sk and begins_with(PK, :pk)"),
		ProjectionExpression:   aws.String("groupID"),
		TableName:              &clients.DynamoTable,
	}
	groupMemberships, queryErr := clients.DynamoClient.Query(input)
	if queryErr != nil {
		logger.Log.Error().Err(queryErr).Str("userId", u.UserID).Msg("error getting users groups")
		return queryErr
	}

	// for each group membership that the user has, leave those groups
	for _, membership := range groupMemberships.Items {
		gm := GroupMember{}
		unMarshErr := dynamodbattribute.UnmarshalMap(membership, &gm)
		if unMarshErr != nil {
			logger.Log.Error().Err(unMarshErr).Str("userID", u.UserID).Msg("error unmarshalling group member to groupMember")
		}
		_, _ = u.LeaveGroup(gm.GroupID)
	}
	return nil
}

// LeaveGroup removes the User from a Group
func (u *User) LeaveGroup(groupID string) (status int, error error) {
	// delete query
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", "GROUP", groupID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, u.UserID)),
			},
		},
		TableName: &clients.DynamoTable,
	}

	// delete from table
	_, err := clients.DynamoClient.DeleteItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", groupID).Str("userID", u.UserID).Msg("Error removing user from group")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("groupID", groupID).Str("userID", u.UserID).Msg("User left group")
	return http.StatusNoContent, nil
}

// CreateToken returns a new CreateToken for the user
func (u *User) CreateToken() (string, error) {
	// create the token
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims = &userClaims{
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
