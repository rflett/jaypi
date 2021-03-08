package clients

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"os"
)

var (
	awsSession, _    = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	SQSClient        = sqs.New(awsSession)
	SNSClient        = sns.New(awsSession)
	DynamoClient     = dynamodb.New(awsSession)
	SecretsClient    = secretsmanager.New(awsSession)
	S3Client         = s3.New(awsSession)
	DynamoTable      = os.Getenv("JAYPI_TABLE")
	JWTSigningSecret = "jjj-api-private-signing-key"
)
