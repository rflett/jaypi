package user

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	logger "jjj.rflett.com/jjj-api/log"
	"jjj.rflett.com/jjj-api/types/guess"
	"net/http"
	"os"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	userTable = os.Getenv("USER_TABLE")
)

// User is a User of the application
type User struct {
	ID        string  `json:"id"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Nickname  string  `json:"nickname"`
	Guesses   []guess.Guess `json:"guesses"`
}

func GetAll(userIDs []string) (error error, status int, users []*User) {
	var dbKeys []map[string]*dynamodb.AttributeValue

	for _, userID := range userIDs {
		av := dynamodb.AttributeValue{
			S: aws.String(userID),
		}
		avMap := map[string]*dynamodb.AttributeValue{
			"id": &av,
		}
		dbKeys = append(dbKeys, avMap)
	}

	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			"users": {
				Keys: dbKeys,
			},
		},
	}

	// getItems
	logger.Log.Info().Msg("getting users from the database")
	result, err := db.BatchGetItem(input)

	// handle errors
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			var responseStatus int
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				responseStatus = http.StatusTooManyRequests
			case dynamodb.ErrCodeResourceNotFoundException:
				responseStatus = http.StatusNotFound
			case dynamodb.ErrCodeRequestLimitExceeded:
				responseStatus = http.StatusTooManyRequests
			case dynamodb.ErrCodeInternalServerError:
				responseStatus = http.StatusInternalServerError
			default:
				responseStatus = http.StatusInternalServerError
			}
			logger.Log.Error().Err(aerr).Msg("error getting users from dynamo")
			return aerr, responseStatus, []*User{}
		} else {
			logger.Log.Error().Err(aerr).Msg("error getting users from dynamo")
			return err, http.StatusInternalServerError, []*User{}
		}
	}

	// unmarshal item into []User
	var response []*User

	for _, av := range result.Responses[userTable] {
		var item User
		_ = dynamodbattribute.UnmarshalMap(av, &item)
		response = append(response, &item)
	}

	return nil, http.StatusOK, response
}
