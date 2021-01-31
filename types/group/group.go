package group

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"net/http"
	"os"
	logger "jjj.rflett.com/jjj-api/log"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	groupTable = os.Getenv("GROUP_TABLE")
)

// Group is a collection of users that can view each others song guesses
type Group struct {
	ID       string   `json:"id"`
	Nickname string   `json:"nickname"`
	Owner    string   `json:"owner"`
	Members  []string `json:"members" dynamodbav:"members,stringset"`
}

func (g *Group) validate() (bool, *string) {
	logger.Log.Debug().Msg("Validating the group")
	// contains the reason for failed validation
	validationMsg := ""

	// check the owner doesn't have a group already
	input := &dynamodb.QueryInput{
		ExpressionAttributeNames: map[string]*string{
			"#O": aws.String("owner"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":o": {
				S: aws.String(g.Owner),
			},
		},
		IndexName: aws.String("owner"),
		KeyConditionExpression: aws.String("#O = :o"),
		ProjectionExpression:   aws.String("id"),
		TableName:              aws.String(groupTable),
	}

	// query dynamo
	result, err := db.Query(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				validationMsg = dynamodb.ErrCodeProvisionedThroughputExceededException
			case dynamodb.ErrCodeResourceNotFoundException:
				validationMsg = dynamodb.ErrCodeResourceNotFoundException
			case dynamodb.ErrCodeRequestLimitExceeded:
				validationMsg = dynamodb.ErrCodeRequestLimitExceeded
			case dynamodb.ErrCodeInternalServerError:
				validationMsg = dynamodb.ErrCodeInternalServerError
			default:
				validationMsg = aerr.Error()
			}
		} else {
			validationMsg = err.Error()
		}
		logger.Log.Error().Err(err).Msg(fmt.Sprint("error querying dynamo during group validation"))
		return false, &validationMsg
	}

	if *result.Count != 0 {
		validationMsg = fmt.Sprintf("group owner %s already has a group", g.Owner)
		return false, &validationMsg
	}

	return true, &validationMsg
}

// Add the Group to the database
func (g *Group) Add() (error error, status int) {
	// validate
	valid, validationMsg := g.validate()
	if !valid {
		return errors.New(*validationMsg), http.StatusBadRequest
	}

	// add the owner as the only original member
	g.Members = []string{g.Owner}

	// create attribute value
	av, _ := dynamodbattribute.MarshalMap(g)

	// create query
	input := &dynamodb.PutItemInput{
		TableName:    aws.String(groupTable),
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}

	// add group to dynamo
	logger.Log.Info().Str("groupOwner", g.Owner).Msg(fmt.Sprintf("adding group to dynamo"))
	_, err := db.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Msg(fmt.Sprintf("error adding group to dynamo %+v", g))
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusCreated
}

// Get a Group from the database
func (g *Group) Get() (error error, status int) {
	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(g.ID),
			},
		},
		TableName: aws.String(groupTable),
	}

	// getItem
	logger.Log.Info().Str("groupID", g.ID).Msg(fmt.Sprintf("getting group from dynamo"))
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
			logger.Log.Error().Err(aerr).Str("groupID", g.ID).Msg("error getting group from dynamo")
			return aerr, responseStatus
		} else {
			logger.Log.Error().Err(aerr).Str("groupID", g.ID).Msg("error getting group from dynamo")
			return err, http.StatusInternalServerError
		}
	}

	if len(result.Item) == 0 {
		return nil, http.StatusNotFound
	}

	// unmarshal item into the Group struct
	err = dynamodbattribute.UnmarshalMap(result.Item, &g)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.ID).Msg("failed to unmarshal dynamo item to group")
	}

	return nil, http.StatusOK
}

// RemoveMember from a Group
func (g *Group) RemoveMember(member string) (error error, status int) {
	// update query
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(groupTable),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(g.ID),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#M": aws.String("members"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":m": {
				SS: []*string{aws.String(member)},
			},
		},
		UpdateExpression: aws.String("DELETE #M :m"),
		ReturnValues:     aws.String("NONE"),
	}

	// update
	logger.Log.Info().Str("groupID", g.ID).Str("groupMember", member).Msg(fmt.Sprintf("removing member from group"))
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
			logger.Log.Error().Err(aerr).Str("groupID", g.ID).Str("groupMember", member).Msg("error removing member from group")
			return aerr, responseStatus
		} else {
			logger.Log.Error().Err(aerr).Str("groupID", g.ID).Str("groupMember", member).Msg("error removing member from group")
			return err, http.StatusInternalServerError
		}
	}

	return nil, http.StatusNoContent
}
