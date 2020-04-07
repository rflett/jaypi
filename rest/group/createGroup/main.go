package main

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

const groupTable = "groups"

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
)

// BodyRequest is the expected body of the create group request
type BodyRequest struct {
	Nickname string `json:"nickname"`
}

// Group is the group DTO
type Group struct {
	ID       string   `json:"id"`
	Nickname string   `json:"nickname"`
	Owner    string   `json:"owner"`
	Members  []string `json:"members"`
}

func createItem(group Group) error {
	// create attribute value
	av, _ := dynamodbattribute.MarshalMap(group)

	// create query
	input := &dynamodb.PutItemInput{
		TableName:    aws.String(groupTable),
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}

	// update
	_, err := db.PutItem(input)

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
		return err
	}
	return nil
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// unmarshall request body to BodyRequest struct
	bodyRequest := BodyRequest{}
	err := json.Unmarshal([]byte(request.Body), &bodyRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, nil
	}

	// create
	userID := "a2972d18-2862-4eff-9af5-49a177455cb5"
	g := Group{
		ID:       "fd4f08ef-99da-4832-ad0c-0e9d2de6af48",
		Nickname: bodyRequest.Nickname,
		Owner:    userID,
		Members:  []string{userID},
	}
	createErr := createItem(g)
	if createErr != nil {
		return events.APIGatewayProxyResponse{Body: createErr.Error(), StatusCode: 500}, nil
	}

	// create and send the response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: 204}, nil
}

func main() {
	lambda.Start(Handler)
}
