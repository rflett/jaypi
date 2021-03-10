package types

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/sns"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
)

type PlatformApp struct {
	Arn      string
	Platform string // Platform is the device platform, either 'ios' or 'android'
}

type PlatformEndpoint struct {
	PK       string `json:"-" dynamodbav:"PK"`
	SK       string `json:"-" dynamodbav:"SK"`
	Arn      string `json:"-" dynamodbav:"arn"`
	UserID   string `json:"-" dynamodbav:"userID"`
	Platform string `json:"-" dynamodbav:"platform"`
}

// GetPlatformEndpointFromToken returns a PlatformEndpoint based on the device token
func (p *PlatformApp) GetPlatformEndpointFromToken(token *string) (platformEndpoint *PlatformEndpoint, err error) {
	input := &sns.ListEndpointsByPlatformApplicationInput{PlatformApplicationArn: &p.Arn}
	err = clients.SNSClient.ListEndpointsByPlatformApplicationPages(input, func(page *sns.ListEndpointsByPlatformApplicationOutput, lastPage bool) bool {
		for _, endpoint := range page.Endpoints {
			if *endpoint.Attributes["Token"] == *token {
				platformEndpoint = &PlatformEndpoint{
					UserID:   *endpoint.Attributes["CustomUserData"],
					Arn:      *endpoint.EndpointArn,
					Platform: p.Platform,
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

// CreatePlatformEndpoint creates a PlatformEndpoint for a user with their token
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

	// create platform endpoint in table
	pe := PlatformEndpoint{
		PK:       fmt.Sprintf("%s#%s", UserPrimaryKey, userID),
		SK:       fmt.Sprintf("%s#%s#%s", EndpointSortKey, p.Platform, *endpoint.EndpointArn),
		UserID:   userID,
		Arn:      *endpoint.EndpointArn,
		Platform: p.Platform,
	}
	av, _ := dynamodbattribute.MarshalMap(pe)
	input := &dynamodb.PutItemInput{
		TableName:    &clients.DynamoTable,
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}
	if _, err = clients.DynamoClient.PutItem(input); err != nil {
		logger.Log.Error().Err(err).Str("platformAppArn", p.Arn).Str("endpointArn", *endpoint.EndpointArn).Msg("Error adding endpoint arn to user")
		return err
	}

	logger.Log.Info().Str("userID", userID).Str("endpointArn", *endpoint.EndpointArn).Msg("Successfully registered SNS endpoint for user")
	return nil
}

// Delete a PlatformEndpoint from SNS and the user's endpoints in dynamo
func (p *PlatformEndpoint) Delete() error {
	// create the endpoint
	snsInput := &sns.DeleteEndpointInput{EndpointArn: &p.Arn}
	_, err := clients.SNSClient.DeleteEndpoint(snsInput)
	if err != nil {
		logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Error deleting endpoint")
		return err
	}

	// delete platform endpoint from table
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", UserPrimaryKey, p.UserID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s#%s", EndpointSortKey, p.Platform, p.Arn)),
			},
		},
		TableName: &clients.DynamoTable,
	}
	if _, err = clients.DynamoClient.DeleteItem(input); err != nil {
		logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Error deleting endpoint arn from user")
		return err
	}

	logger.Log.Info().Str("userID", p.UserID).Str("endpointArn", p.Arn).Msg("Successfully deleted SNS endpoint for user")
	return nil
}

// SendAppleNotification sends a Notification to an ios PlatformEndpoint
func (p *PlatformEndpoint) SendAppleNotification(notification *Notification) error {
	input := &sns.PublishInput{
		Message:          aws.String(notification.IosPayload()),
		MessageStructure: aws.String("json"),
		TargetArn:        &p.Arn,
	}
	message, err := clients.SNSClient.Publish(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == sns.ErrCodeEndpointDisabledException {
				logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Cannot send ios notification because endpoint is disabled")
				return p.Delete()
			}
		}
		logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Error publishing ios notification to SNS")
		return err
	}

	logger.Log.Info().Str("userID", p.UserID).Str("messageID", *message.MessageId).Msg("Successfully sent ios notification")
	return nil
}

// SendAndroidNotification sends a Notification to an ios PlatformEndpoint
func (p *PlatformEndpoint) SendAndroidNotification(notification *Notification) error {
	input := &sns.PublishInput{
		Message:          aws.String(notification.AndroidPayload()),
		MessageStructure: aws.String("json"),
		TargetArn:        &p.Arn,
	}
	message, err := clients.SNSClient.Publish(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == sns.ErrCodeEndpointDisabledException {
				logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Cannot send android notification because endpoint is disabled")
				return p.Delete()
			}
		}
		logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Error publishing android notification to SNS")
	}

	logger.Log.Info().Str("userID", p.UserID).Str("messageID", *message.MessageId).Msg("Successfully sent android notification")
	return nil
}

// SendNotification sends a notification to a PlatformEndpoint
func (p *PlatformEndpoint) SendNotification(n *Notification) error {
	// generate message
	var message string
	switch p.Platform {
	case SNSPlatformGoogle:
		message = n.AndroidPayload()
	case SNSPlatformApple:
		message = n.IosPayload()
	default:
		return errors.New("unsupported platform")
	}

	resp, err := clients.SNSClient.Publish(&sns.PublishInput{
		Message:          &message,
		MessageStructure: aws.String("json"),
		TargetArn:        &p.Arn,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == sns.ErrCodeEndpointDisabledException {
				logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Cannot send android notification because endpoint is disabled")
				return p.Delete()
			}
		}
		logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Error publishing android notification to SNS")
	}

	logger.Log.Info().Str("userID", p.UserID).Str("messageID", *resp.MessageId).Msg("Successfully sent android notification")
	return nil
}
