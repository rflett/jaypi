package types

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/sns"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"time"
)

type PlatformApp struct {
	Arn      string
	Platform string // Platform is the device platform, either 'ios' or 'android'
}

type PlatformEndpoint struct {
	Arn    string
	UserID *string
}

// GetPlatformEndpointFromToken
func (p *PlatformApp) GetPlatformEndpointFromToken(token *string) (platformEndpoint *PlatformEndpoint, err error) {
	input := &sns.ListEndpointsByPlatformApplicationInput{PlatformApplicationArn: aws.String(p.Arn)}
	err = clients.SNSClient.ListEndpointsByPlatformApplicationPages(input, func(page *sns.ListEndpointsByPlatformApplicationOutput, lastPage bool) bool {
		for _, endpoint := range page.Endpoints {
			if *endpoint.Attributes["Token"] == *token {
				platformEndpoint = &PlatformEndpoint{
					UserID: endpoint.Attributes["CustomUserData"],
					Arn:    *endpoint.EndpointArn,
				}
				return false
			}
		}
		return !lastPage
	})
	if err != nil {
		logger.Log.Error().Err(err).Str("platformAppArn", p.Arn).Msg("error listing endpoints for platform app")
		return nil, err
	}
	return platformEndpoint, nil
}

// CreatePlatformEndpoint
func (p *PlatformApp) CreatePlatformEndpoint(userID string, token *string) error {
	// create the endpoint
	snsInput := &sns.CreatePlatformEndpointInput{
		CustomUserData:         &userID,
		PlatformApplicationArn: &p.Arn,
		Token:                  token,
	}
	endpoint, err := clients.SNSClient.CreatePlatformEndpoint(snsInput)
	if err != nil {
		logger.Log.Error().Err(err).Str("platformAppArn", p.Arn).Msg("Error creating platform endpoint")
		return err
	}

	// set fields
	updatedAt := time.Now().UTC().Format(time.RFC3339)

	// update table item query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#PE": aws.String(fmt.Sprintf("%sEndpoints", p.Platform)),
			"#UA": aws.String("updatedAt"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pe": {
				SS: []*string{endpoint.EndpointArn},
			},
			":ua": {
				S: &updatedAt,
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, userID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserSortKey, userID)),
			},
		},
		ReturnValues:     aws.String("NONE"),
		TableName:        &clients.DynamoTable,
		UpdateExpression: aws.String("ADD #PE :pe SET #UA = :ua"),
	}
	_, err = clients.DynamoClient.UpdateItem(input)
	if err != nil {
		logger.Log.Error().Err(err).Str("platformAppArn", p.Arn).Str("endpointArn", *endpoint.EndpointArn).Msg("Error adding endpoint arn to user")
		return err
	}

	logger.Log.Info().Str("userID", userID).Str("endpointArn", *endpoint.EndpointArn).Msg("Successfully registered SNS endpoint for user")
	return nil
}

// DeletePlatformEndpoint
func (p *PlatformApp) DeletePlatformEndpoint(pe *PlatformEndpoint) error {
	// create the endpoint
	snsInput := &sns.DeleteEndpointInput{EndpointArn: &pe.Arn}
	_, err := clients.SNSClient.DeleteEndpoint(snsInput)
	if err != nil {
		logger.Log.Error().Err(err).Str("endpointArn", pe.Arn).Msg("Error deleting endpoint")
		return err
	}

	// set fields
	updatedAt := time.Now().UTC().Format(time.RFC3339)

	// update table item query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#PE": aws.String(fmt.Sprintf("%sEndpoints", p.Platform)),
			"#UA": aws.String("updatedAt"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pe": {
				SS: []*string{&pe.Arn},
			},
			":ua": {
				S: &updatedAt,
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, *pe.UserID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserSortKey, *pe.UserID)),
			},
		},
		ReturnValues:     aws.String("NONE"),
		TableName:        &clients.DynamoTable,
		UpdateExpression: aws.String("DELETE #PE :pe SET #UA = :ua"),
	}
	_, err = clients.DynamoClient.UpdateItem(input)
	if err != nil {
		logger.Log.Error().Err(err).Str("endpointArn", pe.Arn).Msg("Error deleting endpoint arn from user")
		return err
	}

	logger.Log.Info().Str("userID", *pe.UserID).Str("endpointArn", pe.Arn).Msg("Successfully deleted SNS endpoint for user")
	return nil
}
