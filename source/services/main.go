package services

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/sns"
	"golang.org/x/crypto/bcrypt"
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

// GetEndpointInfo paginates the endpoints in a platform application
func GetEndpointInfo(arn *string, token string) (*types.NotieEndpointInfo, error) {
	var userData *string
	input := &sns.ListEndpointsByPlatformApplicationInput{PlatformApplicationArn: arn}
	err := clients.SNSClient.ListEndpointsByPlatformApplicationPages(input, func(page *sns.ListEndpointsByPlatformApplicationOutput, lastPage bool) bool {
		for _, endpoint := range page.Endpoints {
			if *endpoint.Attributes["Token"] == token {
				userData = endpoint.Attributes["CustomUserData"]
				return false
			}
		}
		return !lastPage
	})

	if err != nil {
		logger.Log.Error().Err(err).Str("arn", *arn).Msg("error listing endpoints for platform app")
		return &types.NotieEndpointInfo{}, err
	}
	return nil, nil
}
