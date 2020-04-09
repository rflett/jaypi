package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

const (
	groupTable = "groups"
	userTable  = "users"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
)

// Guess is a song guess consisting of an artist and song name
type Guess struct {
	Artist string `json:"artist"`
	Song   string `json:"song"`
}

// user is a user of the application
type user struct {
	ID        string  `json:"id"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Nickname  string  `json:"nickname"`
	Guesses   []Guess `json:"guesses"`
}

// Group is a group of users that compete against eachother
type Group struct {
	ID       string   `json:"id"`
	Nickname string   `json:"nickname"`
	Owner    string   `json:"owner"`
	Members  []string `json:"members"`
}

func getUsers(userIDs []string) ([]user, error) {
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
	result, err := db.BatchGetItem(input)

	// handle errors
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return []user{}, err
	}

	// unmarshal item into []user
	var users []user

	for _, av := range result.Responses[userTable] {
		var item user
		_ = dynamodbattribute.UnmarshalMap(av, &item)
		users = append(users, item)
	}

	return users, nil

}

func getGroupMemberIDs(groupID string) ([]string, error) {

	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(groupID),
			},
		},
		TableName: aws.String(groupTable),
	}

	// getItem
	result, err := db.GetItem(input)

	// handle errors
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return []string{""}, err
	}

	// unmarshal item into the user struct
	item := Group{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		fmt.Printf("Failed to unmarshal Record, %v", err)
		return []string{""}, errors.New("Failed to unmarshall Record to Group")
	}

	return item.Members, nil
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	memberIDs, membersErr := getGroupMemberIDs(groupID)
	if membersErr != nil {
		return events.APIGatewayProxyResponse{Body: membersErr.Error(), StatusCode: 400}, nil
	}

	users, usersErr := getUsers(memberIDs)
	if usersErr != nil {
		return events.APIGatewayProxyResponse{Body: usersErr.Error(), StatusCode: 400}, nil
	}

	// create and send the response
	body, _ := json.Marshal(users)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: 200, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
