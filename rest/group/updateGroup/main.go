package main

import (
	"encoding/json"
	"fmt"

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

// BodyRequest is the expected request body
type BodyRequest struct {
	Nickname string `json:"nickname"`
	Owner    string `json:"owner"`
}

func updateItem(groupID string, body BodyRequest) error {
	// update query
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(groupTable),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(groupID),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#N": aws.String("nickname"),
			"#O": aws.String("owner"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":n": {
				S: aws.String(body.Nickname),
			},
			":o": {
				S: aws.String(body.Owner),
			},
		},
		UpdateExpression: aws.String("SET #N = :n, #O = :o"),
		ReturnValues:     aws.String("NONE"),
	}

	// update
	_, err := db.UpdateItem(input)

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

	// get groupID from pathParameters
	groupID := request.PathParameters["groupId"]

	// unmarshal request body to BodyRequest struct
	bodyRequest := BodyRequest{}
	err := json.Unmarshal([]byte(request.Body), &bodyRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, nil
	}

	// update
	updateErr := updateItem(groupID, bodyRequest)
	if updateErr != nil {
		return events.APIGatewayProxyResponse{Body: updateErr.Error(), StatusCode: 500}, nil
	}

	// create and send the response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: 204}, nil
}

func main() {
	lambda.Start(Handler)
}
