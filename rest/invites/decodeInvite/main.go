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

const inviteLink = "https://www.fuckYouNorto.com/invite?groupCode=%s"

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
)

// ResponseDto is the response that we'll send back
type ResponseDto struct {
	GroupID string `json:"id"`
}

func getUserGroupID(userID string) (ResponseDto, error) {
	// scan is really bad and we need to think of a better way to do this
	// maybe store the groupID on the user?
	input := &dynamodb.ScanInput{
		ExpressionAttributeNames: map[string]*string{
			"#I": aws.String("id"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":m": {
				S: aws.String(userID),
			},
		},
		FilterExpression:     aws.String("contains (members, :m)"),
		ProjectionExpression: aws.String("#I"),
		TableName:            aws.String("groups"),
	}

	result, err := db.Scan(input)
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
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return ResponseDto{}, err
	}

	// unmarshal groupID into the Group struct
	groupID := ResponseDto{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &groupID)
	if err != nil {
		fmt.Printf("Failed to unmarshal Record, %v", err)
		return ResponseDto{}, errors.New("Failed to unmarshall Record to ResponseDto")
	}

	return groupID, nil
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// get groupCode from request
	code := request.QueryStringParameters["groupCode"]

	// get the user's group id
	responseDto, _ := getUserGroupID(code)

	// create and send the response
	body, _ := json.Marshal(responseDto)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: 200, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
