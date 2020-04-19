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

const userTable = "users"

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
)

// BodyRequest is the expected body of the update user guesses request
type BodyRequest struct {
	Guesses []string `json:"guesses" dynamodbav:"guesses,stringset"`
}

func updateItem(userID string, body BodyRequest) error {
	var songSS []*string
	for _, songID := range body.Guesses {
		songSS = append(songSS, aws.String(songID))
	}

	// update query
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(userTable),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(userID),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#G": aws.String("guesses"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":g": {
				SS: songSS,
			},
		},
		UpdateExpression: aws.String("SET #G = :g"),
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

	// get userId from pathParameters
	userID := request.PathParameters["userId"]

	// unmarshall request body to BodyRequest struct
	bodyRequest := BodyRequest{}
	err := json.Unmarshal([]byte(request.Body), &bodyRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, nil
	}

	// update
	updateErr := updateItem(userID, bodyRequest)
	if updateErr != nil {
		return events.APIGatewayProxyResponse{Body: updateErr.Error(), StatusCode: 500}, nil
	}

	// create and send the response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: 204}, nil
}

func main() {
	lambda.Start(Handler)
}
