package clients

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	awsConfig, _  = config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("countdown"), config.WithRegion("ap-southeast-2"))
	SQSClient        = sqs.New(awsSession)
	SNSClient        = sns.New(awsSession)
	DynamoClient     = dynamodb.NewFromConfig(awsConfig)
	SecretsClient    = secretsmanager.New(awsSession)
	S3Client         = s3.New(awsSession)
	DynamoTable      = "jaypi-staging"
	JWTSigningSecret = "jaypi-private-key-staging"
)
