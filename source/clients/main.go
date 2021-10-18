package clients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var (
	awsConfig, _     = config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("countdown"), config.WithRegion("ap-southeast-2"))
	SQSClient        = sqs.NewFromConfig(awsConfig)
	SNSClient        = sns.NewFromConfig(awsConfig)
	DynamoClient     = dynamodb.NewFromConfig(awsConfig)
	SecretsClient    = secretsmanager.NewFromConfig(awsConfig)
	S3Client         = s3.NewFromConfig(awsConfig)
	DynamoTable      = "jaypi-staging"
	JWTSigningSecret = "jaypi-private-key-staging"
)
