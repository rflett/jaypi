package playCount

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	logger "jjj.rflett.com/jjj-api/log"
	"os"
)

const (
	PrimaryKey = "PLAYCOUNT"
	SortKey    = "CURRENT"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	table         = os.Getenv("JAYPI_TABLE")
)

type songCount struct {
	Value *string `json:"value"`
}

// GetCurrentPlayCount looks up the current songCount item and returns its value
func GetCurrentPlayCount() (*string, error) {
	input := &dynamodb.QueryInput{
		ExpressionAttributeNames: map[string]*string{
			"#V": aws.String("value"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(PrimaryKey),
			},
			":sk": {
				S: aws.String(SortKey),
			},
		},
		KeyConditionExpression: aws.String("SK = :sk and PK = :pk"),
		ProjectionExpression:   aws.String("#V"),
		TableName:              &table,
	}
	result, err := db.Query(input)
	if err != nil || *result.Count == 0 {
		logger.Log.Error().Err(err).Msg("Unable to get the latest song position")
		return aws.String("0"), err
	}

	var sc = songCount{}
	unmarshalErr := dynamodbattribute.UnmarshalMap(result.Items[0], &sc)
	if unmarshalErr != nil {
		logger.Log.Error().Err(unmarshalErr).Msg("Unable to unmarshall query result to songCount")
		return aws.String("0"), unmarshalErr
	}
	return sc.Value, nil
}

// IncrementPlayCount increments the current songCount value
func IncrementPlayCount() {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#V": aws.String("value"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":inc": {
				N: aws.String("1"),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(PrimaryKey),
			},
			"SK": {
				S: aws.String(SortKey),
			},
		},
		ReturnValues:     aws.String("NONE"),
		TableName:        aws.String("jaypi"),
		UpdateExpression: aws.String("ADD #V :inc"),
	}
	_, err := db.UpdateItem(input)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to increment the latest song position")
	}
}
