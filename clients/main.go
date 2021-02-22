package clients

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/sqs"
	"os"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	SQSClient     = sqs.New(awsSession)
	DynamoClient  = dynamodb.New(awsSession)
	DynamoTable   = os.Getenv("JAYPI_TABLE")
)
