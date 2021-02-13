package groupCode

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dchest/uniuri"
	logger "jjj.rflett.com/jjj-api/log"
	"os"
)

const (
	PrimaryKey = "GROUP"
	SortKey    = "#CODE"
	SecondaryIndex = "GSI1"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	table    = os.Getenv("JAYPI_TABLE")
)

type GroupCode struct {
	PK      string `json:"-" dynamodbav:"PK"`
	SK      string `json:"-" dynamodbav:"SK"`
	GroupID string `json:"groupID"`
	Code    string `json:"code"`
}

// validateCode checks if a code already exists against a group and returns an error if it does
func validateCode(code string) error {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", SortKey, code)),
			},
			":pk": {
				S: aws.String(fmt.Sprintf("%s#", PrimaryKey)),
			},
		},
		IndexName: aws.String(SecondaryIndex),
		KeyConditionExpression: aws.String("SK = :sk and begins_with(PK, :pk)"),
		ProjectionExpression:   aws.String("code"),
		TableName:              aws.String(table),
	}

	// query
	result, err := db.Query(input)

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

// New creates a new code for the group
func New(groupID string) (GroupCode, error) {
	code := ""

	// attempt to create the code
	for i := 1; i <= 5; i++ {
		codeAttempt := uniuri.NewLen(6)
		if ok := validateCode(codeAttempt); ok == nil {
			code = codeAttempt
			break
		}
	}

	// return the error if we couldn't create the code
	if code == "" {
		newCodeError := errors.New("unable to generate new code")
		logger.Log.Error().Err(newCodeError).Str("groupID", groupID)
		return GroupCode{}, newCodeError
	}

	groupCode := GroupCode{
		PK:      fmt.Sprintf("%s#%s", PrimaryKey, groupID),
		SK:      fmt.Sprintf("%s#%s", SortKey, code),
		GroupID: groupID,
		Code:    code,
	}

	// add the code to the table
	av, _ := dynamodbattribute.MarshalMap(groupCode)
	input := &dynamodb.PutItemInput{
		TableName:    aws.String(table),
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}
	_, err := db.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", groupID).Msg("Error adding code to table")
		return GroupCode{}, err
	}

	// success
	logger.Log.Info().Str("groupID", groupID).Msg("Successfully added code to table")
	return groupCode, nil
}

// GetGroupFromCode returns the groupID based on the group code
func GetGroupFromCode(code string) (string, error) {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", SortKey, code)),
			},
			":pk": {
				S: aws.String(fmt.Sprintf("%s#", PrimaryKey)),
			},
		},
		IndexName: aws.String(SecondaryIndex),
		KeyConditionExpression: aws.String("SK = :sk and begins_with(PK, :pk)"),
		ProjectionExpression:   aws.String("groupID"),
		TableName:              aws.String(table),
	}

	// query
	result, err := db.Query(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("code", code).Msg("error querying code")
		return "", err
	}

	// code doesn't exist
	if len(result.Items) == 0 {
		codeNotExistErr := errors.New("code does not exist")
		logger.Log.Error().Err(codeNotExistErr).Str("code", code).Msg("code does not exist")
		return "", codeNotExistErr
	}

	// unmarshal groupID into the Group struct
	gc := GroupCode{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &gc)
	if err != nil {
		logger.Log.Error().Err(err).Str("code", code).Msg("error unmarshalling code to GroupCode")
		fmt.Printf("Failed to unmarshal Record, %v", err)
		return "", err
	}
	return gc.GroupID, nil
}

// Get the group code
func Get(groupId string) (string, error) {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, groupId)),
			},
			":sk": {
				S: aws.String(SortKey),
			},
		},
		KeyConditionExpression: aws.String("PK = :pk and begins_with(SK, :sk)"),
		ProjectionExpression:   aws.String("code"),
		TableName:              aws.String(table),
	}

	// query
	result, err := db.Query(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupId", groupId).Msg("error querying group for code")
		return "", err
	}

	// code doesn't exist
	if len(result.Items) == 0 {
		logger.Log.Info().Str("groupId", groupId).Msg("groupId does not exist")
		return "", nil
	}

	// unmarshal groupID into the Group struct
	gc := GroupCode{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &gc)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupId", groupId).Msg("error unmarshalling groupId to GroupCode")
		fmt.Printf("Failed to unmarshal Record, %v", err)
		return "", err
	}
	return gc.Code, nil
}
