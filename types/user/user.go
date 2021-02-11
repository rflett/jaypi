package user

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	logger "jjj.rflett.com/jjj-api/log"
	"net/http"
	"os"
	"time"
)

const (
	PrimaryKey = "USER"
	SortKey    = "#PROFILE"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	table         = os.Getenv("JAYPI_TABLE")
)

// User is a User of the application
type User struct {
	PK        string  `json:"-" dynamodbav:"PK"`
	SK        string  `json:"-" dynamodbav:"SK"`
	UserID    string  `json:"userID"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	NickName  string  `json:"nickName"`
	Email     *string `json:"email"`
	Points    int     `json:"points"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt *string `json:"updatedAt"`
}

// Create the user and save them to the database
func (u *User) Create() (status int, error error) {
	// set fields
	u.UserID = uuid.NewString()
	u.PK = fmt.Sprintf("%s#%s", PrimaryKey, u.UserID)
	u.SK = fmt.Sprintf("%s#%s", SortKey, u.UserID)
	u.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// create item
	av, _ := dynamodbattribute.MarshalMap(u)

	// create input
	input := &dynamodb.PutItemInput{
		TableName:    aws.String(table),
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}

	// add to table
	_, err := db.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("Error adding user to table")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("userID", u.UserID).Msg("Successfully added user to table")
	return http.StatusCreated, nil
}

// Update the user's attributes
func (u *User) Update() (status int, error error) {
	// set fields
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	u.UpdatedAt = &updatedAt

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#NN": aws.String("nickName"),
			"#UA": aws.String("updatedAt"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":nn": {
				S: aws.String(u.NickName),
			},
			":ua": {
				S: aws.String(*u.UpdatedAt),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, u.UserID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", SortKey, u.UserID)),
			},
		},
		ReturnValues:     aws.String("NONE"),
		TableName:        aws.String(table),
		UpdateExpression: aws.String("SET #NN = :nn, #UA = :ua"),
	}

	_, err := db.UpdateItem(input)

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
			logger.Log.Error().Err(aerr).Str("userID", u.UserID).Msg("error updating user")
			return responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error updating user")
			return http.StatusInternalServerError, err
		}
	}

	return http.StatusNoContent, nil
}

// Get the user from the table
func Get(userID string) (user User, status int, error error) {
	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, userID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", SortKey, userID)),
			},
		},
		TableName: aws.String(table),
	}

	// getItem
	result, err := db.GetItem(input)

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
			logger.Log.Error().Err(aerr).Str("userID", userID).Msg("error getting user from table")
			return User{}, responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("userID", userID).Msg("error getting user from table")
			return User{}, http.StatusInternalServerError, err
		}
	}

	if len(result.Item) == 0 {
		return User{}, http.StatusNotFound, nil
	}

	// unmarshal item into struct
	err = dynamodbattribute.UnmarshalMap(result.Item, &user)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", userID).Msg("failed to unmarshal dynamo item to user")
	}

	return user, http.StatusOK, nil
}
