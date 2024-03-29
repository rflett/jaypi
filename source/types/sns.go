package types

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go/aws"
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
	Arn      string `json:"-"`
	UserID   string `json:"-"`
	Platform string `json:"-"`
}

// GetPlatformEndpointFromToken returns a PlatformEndpoint based on the device token
func (p *PlatformApp) GetPlatformEndpointFromToken(token *string) (platformEndpoint *PlatformEndpoint, err error) {
	input := &sns.ListEndpointsByPlatformApplicationInput{PlatformApplicationArn: &p.Arn}

	paginator := sns.NewListEndpointsByPlatformApplicationPaginator(clients.SNSClient, input)

	for paginator.HasMorePages() && platformEndpoint == nil {
		page, pageErr := paginator.NextPage(context.TODO())
		if pageErr != nil {
			logger.Log.Error().Err(pageErr).Msg("error getting NextPage from GetPlatformEndpointFromToken paginator")
			return nil, pageErr
		}

		for _, endpoint := range page.Endpoints {
			if endpoint.Attributes["Token"] == *token {
				platformEndpoint = &PlatformEndpoint{
					UserID:   endpoint.Attributes["CustomUserData"],
					Arn:      *endpoint.EndpointArn,
					Platform: p.Platform,
				}
				break
			}
		}
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
	endpoint, err := clients.SNSClient.CreatePlatformEndpoint(context.TODO(), snsInput)
	if err != nil {
		logger.Log.Error().Err(err).Str("platformAppArn", p.Arn).Msg("Error creating platform endpoint")
		return err
	}

	// create platform endpoint in table
	pe := PlatformEndpoint{
		PK:       fmt.Sprintf("%s#%s", UserPartitionKey, userID),
		SK:       fmt.Sprintf("%s#%s#%s", EndpointSortKey, p.Platform, *endpoint.EndpointArn),
		UserID:   userID,
		Arn:      *endpoint.EndpointArn,
		Platform: p.Platform,
	}
	av, _ := attributevalue.MarshalMap(pe)
	input := &dynamodb.PutItemInput{
		TableName:    &DynamoTable,
		Item:         av,
		ReturnValues: dbTypes.ReturnValueNone,
	}
	if _, err = clients.DynamoClient.PutItem(context.TODO(), input); err != nil {
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
	_, err := clients.SNSClient.DeleteEndpoint(context.TODO(), snsInput)
	if err != nil {
		logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Error deleting endpoint")
		return err
	}

	// delete platform endpoint from table
	input := &dynamodb.DeleteItemInput{
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: fmt.Sprintf("%s#%s", UserPartitionKey, p.UserID)},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: fmt.Sprintf("%s#%s#%s", EndpointSortKey, p.Platform, p.Arn)},
		},
		TableName: &DynamoTable,
	}
	if _, err = clients.DynamoClient.DeleteItem(context.TODO(), input); err != nil {
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
	message, err := clients.SNSClient.Publish(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Error publishing ios notification to SNS")
		return p.Delete()
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
	message, err := clients.SNSClient.Publish(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Error publishing android notification to SNS")
		return p.Delete()
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

	resp, err := clients.SNSClient.Publish(context.TODO(), &sns.PublishInput{
		Message:          &message,
		MessageStructure: aws.String("json"),
		TargetArn:        &p.Arn,
	})
	if err != nil {
		logger.Log.Error().Err(err).Str("endpointArn", p.Arn).Msg("Error publishing android notification to SNS")
		return p.Delete()
	}

	logger.Log.Info().Str("userID", p.UserID).Str("messageID", *resp.MessageId).Msg("Successfully sent android notification")
	return nil
}
