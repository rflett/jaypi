package group

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	logger "jjj.rflett.com/jjj-api/log"
	"jjj.rflett.com/jjj-api/types/groupCode"
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
	Code      string  `json:"code" dynamodbav:"-"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt *string `json:"updatedAt"`
}

// groupMember is a member of a group
type groupMember struct {
	PK       string `json:"-" dynamodbav:"PK"`
	SK       string `json:"-" dynamodbav:"SK"`
	UserID   string `json:"userID"`
	GroupID  string `json:"groupID"`
	JoinedAt string `json:"joinedAt"`
}

// Create the group and save it to the database
func (g *Group) Create() (status int, error error) {
	// set fields
	g.GroupID = uuid.NewString()
	g.PK = fmt.Sprintf("%s#%s", PrimaryKey, g.GroupID)
	g.SK = fmt.Sprintf("%s#%s", SortKey, g.GroupID)
	g.CreatedAt = time.Now().UTC().Format(time.RFC3339)

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

	// create code
	code, codeErr := groupCode.New(g.GroupID)
	if codeErr != nil {
		return http.StatusInternalServerError, codeErr
	}
	g.Code = code.Code

	// add the owner as a member
	_, joinStatus, joinErr := Join(g.OwnerID, g.Code)
	if joinErr != nil {
		logger.Log.Error().Err(joinErr).Str("groupID", g.GroupID).Msg("Unable to join owner to group")
		return joinStatus, joinErr
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

// Get the group from the table
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

	// get the group code
	code, codeErr := groupCode.Get(groupID)
	if codeErr != nil {
		return group, http.StatusOK, nil
	}
	group.Code = code
	return group, http.StatusOK, nil
}

// Join a user to a group
func Join(userID string, code string) (group Group, status int, error error) {
	groupID, groupIDErr := groupCode.GetGroupFromCode(code)
	if groupIDErr != nil {
		return Group{}, http.StatusBadRequest, groupIDErr
	}

	gm := groupMember{
		PK:       fmt.Sprintf("%s#%s", PrimaryKey, groupID),
		SK:       fmt.Sprintf("%s#%s", "USER", userID),
		UserID:   userID,
		GroupID:  groupID,
		JoinedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// create item
	av, _ := dynamodbattribute.MarshalMap(gm)

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
		logger.Log.Error().Err(err).Str("groupID", groupID).Str("userID", userID).Msg("Error adding user to group")
		return Group{}, http.StatusInternalServerError, err
	}

	// return the group
	logger.Log.Info().Str("groupID", groupID).Str("userID", userID).Msg("Successfully added user to group")
	g, _, getGroupErr := Get(groupID)
	if getGroupErr != nil {
		return Group{}, http.StatusInternalServerError, err
	}
	return g, http.StatusOK, nil
}
