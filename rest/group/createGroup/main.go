package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	groupTable = os.Getenv("GROUP_TABLE")
)

// BodyRequest is the expected body of the create group request
type BodyRequest struct {
	Nickname string `json:"nickname"`
	Owner    string `json:"owner"`
}

// Group is the group DTO
type Group struct {
	ID       string   `json:"id"`
	Nickname string   `json:"nickname"`
	Owner    string   `json:"owner"`
	Members  []string `json:"members" dynamodbav:"members,stringset"`
}

func (g *Group) validate() error {
	// check that the owner doesn't already have a group
	// check the nickname is valid
	return nil
}

func (g *Group) create() error {
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
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// create
	g := Group{
		ID:       uuid.NewString(),
		Nickname: bodyRequest.Nickname,
		Owner:    bodyRequest.Owner,
	}
	validationErr := g.validate()
	if validationErr != nil {
		return events.APIGatewayProxyResponse{Body: validationErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	createErr := g.create()
	if createErr != nil {
		return events.APIGatewayProxyResponse{Body: createErr.Error(), StatusCode: http.StatusInternalServerError}, nil
	}

	// create and send the response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
