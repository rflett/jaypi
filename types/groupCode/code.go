package groupCode

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dchest/uniuri"
	logger "jjj.rflett.com/jjj-api/log"
	"os"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	codesTable    = os.Getenv("GROUP_CODE_TABLE")
)

type GroupCode struct {
	Code    string `json:"code"`
	GroupID string `json:"groupId"`
}

type GetCodeResponse struct {
	GroupID string `json:"groupId"`
}

func (c *GroupCode) exists() (bool, error) {
	code, err := c.Get()
	return code != "", err
}

func (c *GroupCode) saveToDB() error {
	// create attribute value
	av, _ := dynamodbattribute.MarshalMap(c)

	// create query
	input := &dynamodb.PutItemInput{
		TableName:    aws.String(codesTable),
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}

	// add group to dynamo
	logger.Log.Info().Str("code", c.Code).Str("groupId", c.GroupID).Msg(fmt.Sprintf("adding code to dynamo"))
	_, err := db.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Msg(fmt.Sprintf("error adding code to dynamo %+v", c))
		return err
	}
	return nil
}

func (c *GroupCode) Get() (string, error) {
	// get query
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":c": {
				S: aws.String(c.Code),
			},
		},
		KeyConditionExpression: aws.String("code = :c"),
		ProjectionExpression:   aws.String("groupId"),
		TableName:              aws.String(codesTable),
	}

	result, err := db.Query(input)
	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupCode", c.Code).Msg("error getting groupCode from dynamo")
		return "", err
	}

	if len(result.Items) == 0 {
		logger.Log.Info().Str("groupCode", c.Code).Msg("the groupCode does not exist in dynamo")
		return "", nil
	} else {
		logger.Log.Info().Str("groupCode", c.Code).Msg("found groupCode in dynamo")

		// unmarshal item into the GetCodeResponse struct
		var r GetCodeResponse
		err = dynamodbattribute.UnmarshalMap(result.Items[0], &r)
		if err != nil {
			logger.Log.Error().Err(err).Str("groupID", r.GroupID).Msg("failed to unmarshal dynamo item to GetCodeResponse")
		}

		return r.GroupID, nil
	}
}

func (c *GroupCode) New() (string, error) {
	for i := 1; i <= 5; i++ {
		// generate a new groupCode
		c.Code = uniuri.NewLen(6)

		// check if it exists already
		exists, err := c.exists()

		if err == nil && !exists {
			// save it to the DB
			saveErr := c.saveToDB()
			if saveErr != nil {
				return "", saveErr
			}

			// return the groupCode
			return c.Code, nil

		} else if err != nil {
			return "", err
		}
	}

	// complain that it took to much effort to generate the groupCode
	err := fmt.Errorf("it took FIVE attempts to get a groupCode that wasn't in use")
	logger.Log.Error().Err(err).Msg("failed to generate new groupCode")
	return "", err
}
