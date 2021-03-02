package types

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
)

type PlatformApp struct {
	Arn      string
	Platform string
}

// GetUserIDFromToken
func (p *PlatformApp) GetUserIDFromToken(token *string) (userID *string, error error) {
	input := &sns.ListEndpointsByPlatformApplicationInput{PlatformApplicationArn: aws.String(p.Arn)}
	err := clients.SNSClient.ListEndpointsByPlatformApplicationPages(input, func(page *sns.ListEndpointsByPlatformApplicationOutput, lastPage bool) bool {
		for _, endpoint := range page.Endpoints {
			if *endpoint.Attributes["Token"] == *token {
				userID = endpoint.Attributes["CustomUserData"]
				return false
			}
		}
		return !lastPage
	})
	if err != nil {
		logger.Log.Error().Err(err).Str("platformAppArn", p.Arn).Msg("error listing endpoints for platform app")
		return nil, err
	}
	return userID, nil
}

// CreatePlatformEndpoint
func (p *PlatformApp) CreatePlatformEndpoint(token *string, userID *string) error {
	snsInput := &sns.CreatePlatformEndpointInput{
		CustomUserData:         userID,
		PlatformApplicationArn: aws.String(p.Arn),
		Token:                  token,
	}
	endpoint, err := clients.SNSClient.CreatePlatformEndpoint(snsInput)
	if err != nil {
		logger.Log.Error().Err(err).Str("platformAppArn", p.Arn).Msg("error creating platform endpoint")
		return err
	}

	logger.Log.Info().Msg(fmt.Sprintf("Endpoint arn is %s", *endpoint.EndpointArn))
	return nil
}
