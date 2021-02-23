package services

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"golang.org/x/crypto/scrypt"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/types"
)

// GetGroupFromCode returns the groupID based on the group code
func GetGroupFromCode(code string) (*types.Group, error) {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", types.GroupCodeSortKey, code)),
			},
			":pk": {
				S: aws.String(fmt.Sprintf("%s#", types.GroupCodePrimaryKey)),
			},
		},
		IndexName:              aws.String(types.GSI),
		KeyConditionExpression: aws.String("SK = :sk and begins_with(PK, :pk)"),
		ProjectionExpression:   aws.String("groupID"),
		TableName:              &clients.DynamoTable,
	}

	// query
	result, err := clients.DynamoClient.Query(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("code", code).Msg("error querying code")
		return &types.Group{}, err
	}

	// code doesn't exist
	if len(result.Items) == 0 {
		codeNotExistErr := errors.New("code does not exist")
		logger.Log.Error().Err(codeNotExistErr).Str("code", code).Msg("code does not exist")
		return &types.Group{}, codeNotExistErr
	}

	// unmarshal groupID into the Group struct
	gc := types.GroupCode{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &gc)
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

// Hashes a password
func HashPassword(password string, salt []byte) (string, error) {
	hash, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 64)

	if err != nil {
		logger.Log.Error().Msg(fmt.Sprintf("failed to hash a password: %v", err))
		return "", err
	}

	return hex.EncodeToString(hash), nil
}

// Check whether a plaintext password matches a hashed one
func PasswordsMatch(password string, salt string, hashedPassword string) bool {
	hashedPassword, err := HashPassword(password, []byte(salt))

	if err != nil {
		// User can't do anything about this so log to the developer and generate a password that'll never be matched
		logger.Log.Error().Msg(fmt.Sprintf("Failed to match a password: %v", err))
		hashedPassword = uuid.NewString()
	}

	return hashedPassword == hashedPassword
}

func GetOauthProvider(provider string) {
	provider, exists := oauthResponses.OauthProviders[providerName]

	if !exists {
		return nil, fmt.Errorf("Sorry. That auth provider isn't supported")
	}

	return provider, nil
}
