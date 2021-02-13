package group

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dchest/uniuri"
	"github.com/google/uuid"
	logger "jjj.rflett.com/jjj-api/log"
	"net/http"
	"os"
	"time"
)

const (
	PrimaryKey = "GROUP"
	SortKey    = "#PROFILE"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	table         = os.Getenv("JAYPI_TABLE")
)

// Group is way for users to be associated with each other
type Group struct {
	PK        string  `json:"-" dynamodbav:"PK"`
	SK        string  `json:"-" dynamodbav:"SK"`
	GroupID   string  `json:"groupID"`
	OwnerID   string  `json:"ownerID"`
	Name      string  `json:"name"`
	Code      string  `json:"code"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt *string `json:"updatedAt"`
}

// validateCode checks if a code already exists against a group and returns an error if it does
func validateCode(code string) error {
	// input
	input := &dynamodb.ScanInput{
		ExpressionAttributeNames: map[string]*string{
			"#C": aws.String("code"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(PrimaryKey),
			},
			":sk": {
				S: aws.String(SortKey),
			},
			":c": {
				S: aws.String(code),
			},
		},
		FilterExpression:     aws.String("begins_with(PK, :pk) and begins_with(SK, :sk) and #C = :c"),
		ProjectionExpression: aws.String("PK"),
		TableName:            aws.String(table),
	}

	// query
	result, err := db.Scan(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("code", code).Msg("error validating code")
		return err
	}

	// code doesn't exist
	if len(result.Items) == 0 {
		logger.Log.Info().Str("code", code).Msg("code does not exist")
		return nil
	}

	// code exists
	logger.Log.Info().Str("code", code).Msg("code already exists")
	return errors.New("code already exists")
}

// newCode creates a new code for the group
func (g *Group) newCode() error {
	for i := 1; i <= 5; i++ {
		codeAttempt := uniuri.NewLen(6)

		err := validateCode(codeAttempt)
		if err == nil {
			g.Code = codeAttempt
			return nil
		}
	}
	newCodeError := errors.New("unable to generate new code")
	logger.Log.Error().Err(newCodeError).Str("groupID", g.GroupID).Msg("Cannot set new code on group")
	return newCodeError
}

// Create the group and save it to the database
func (g *Group) Create() (status int, error error) {
	// set fields
	g.GroupID = uuid.NewString()
	g.PK = fmt.Sprintf("%s#%s", PrimaryKey, g.GroupID)
	g.SK = fmt.Sprintf("%s#%s", SortKey, g.GroupID)
	g.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	codeErr := g.newCode()
	if codeErr != nil {
		return http.StatusInternalServerError, codeErr
	}

	// create item
	av, _ := dynamodbattribute.MarshalMap(g)

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
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("Error adding group to table")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("groupID", g.GroupID).Msg("Successfully added group to table")
	return http.StatusCreated, nil
}

// Update the group's attributes
func (g *Group) Update() (status int, error error) {
	// set fields
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	g.UpdatedAt = &updatedAt

	pk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, g.GroupID)),
	}
	sk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", SortKey, g.GroupID)),
	}

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#N":  aws.String("name"),
			"#UA": aws.String("updatedAt"),
			"#O":  aws.String("ownerID"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": &pk,
			":sk": &sk,
			":ua": {
				S: aws.String(*g.UpdatedAt),
			},
			":n": {
				S: aws.String(g.Name),
			},
			":o": {
				S: aws.String(g.OwnerID),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": &pk,
			"SK": &sk,
		},
		ReturnValues:        aws.String("NONE"),
		TableName:           aws.String(table),
		ConditionExpression: aws.String("PK = :pk and SK = :sk and #O = :o"),
		UpdateExpression:    aws.String("SET #N = :n, #UA = :ua"),
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
			case dynamodb.ErrCodeConditionalCheckFailedException:
				responseStatus = http.StatusNotFound
			case dynamodb.ErrCodeRequestLimitExceeded:
				responseStatus = http.StatusTooManyRequests
			case dynamodb.ErrCodeInternalServerError:
				responseStatus = http.StatusInternalServerError
			default:
				responseStatus = http.StatusInternalServerError
			}
			logger.Log.Error().Err(aerr).Str("groupID", g.GroupID).Msg("error updating group")
			return responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error updating group")
			return http.StatusInternalServerError, err
		}
	}

	return http.StatusNoContent, nil
}

// Get the user from the table
func Get(groupID string) (group Group, status int, error error) {
	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, groupID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", SortKey, groupID)),
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
			logger.Log.Error().Err(aerr).Str("groupID", groupID).Msg("error getting group from table")
			return Group{}, responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("groupID", groupID).Msg("error getting group from table")
			return Group{}, http.StatusInternalServerError, err
		}
	}

	if len(result.Item) == 0 {
		return Group{}, http.StatusNotFound, nil
	}

	// unmarshal item into struct
	err = dynamodbattribute.UnmarshalMap(result.Item, &group)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", groupID).Msg("failed to unmarshal dynamo item to group")
	}

	return group, http.StatusOK, nil
}
