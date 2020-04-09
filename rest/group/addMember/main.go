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

// RequestDto is the expected request body
type RequestDto struct {
	Code    string `json:"code"`
	GroupID string `json:"groupId"`
	UserID  string `json:"userId"`
}

func addUserToGroup(userID, groupID string) error {
	// update query
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(groupTable),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(groupID),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#M": aws.String("members"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":m": {
				SS: []*string{aws.String(userID)},
			},
		},
		UpdateExpression: aws.String("ADD #M :m"),
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

	// unmarshall request body to RequestDto struct
	requestDto := RequestDto{}
	err := json.Unmarshal([]byte(request.Body), &requestDto)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, nil
	}

	// add userId to group
	updateErr := addUserToGroup(requestDto.UserID, requestDto.GroupID)
	if updateErr != nil {
		return events.APIGatewayProxyResponse{Body: updateErr.Error(), StatusCode: 500}, nil
	}

	// create and send the response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: 204}, nil
}

func main() {
	lambda.Start(Handler)
}
