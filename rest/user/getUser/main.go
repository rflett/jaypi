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

const userTable = "users"

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
)

// Song is a song Song consisting of an artist and song name
type Song struct {
	ID     string `json:"id"`
	Artist string `json:"artist"`
	Title  string `json:"title"`
}

// user is a user of the application
type user struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Nickname  string `json:"nickname"`
	Guesses   []Song `json:"guesses"`
}

// Item is a db item
type Item struct {
	ID        string   `json:"id"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Nickname  string   `json:"nickname"`
	Guesses   []string `json:"guesses" dynamodbav:"guesses,stringset"`
}

func getSongs(SongIDs []string) []Song {
	songs := []Song{
		Song{
			ID:     "1",
			Artist: "Dream Theater",
			Title:  "Raise The Knife",
		},
		Song{
			ID:     "2",
			Artist: "Opeth",
			Title:  "When",
		},
		Song{
			ID:     "3",
			Artist: "Bloodbath",
			Title:  "Eaten",
		},
	}

	var ret []Song
	for idx := range SongIDs {
		ret = append(ret, songs[idx])
	}
	return ret
}

func getItem(userID string) (user, error) {

	// create query
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(userID),
			},
		},
		TableName: aws.String(userTable),
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
		return user{}, err
	}

	// unmarshal item into the user struct
	item := Item{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		fmt.Printf("Failed to unmarshal Record, %v", err)
		return user{}, errors.New("Failed to unmarshall Record to user")
	}

	songs := getSongs(item.Guesses)
	thisUser := user{
		ID:        item.ID,
		FirstName: item.FirstName,
		LastName:  item.LastName,
		Nickname:  item.Nickname,
		Guesses:   songs,
	}

	return thisUser, nil
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// get userId from pathParameters
	userID := request.PathParameters["userId"]

	user, err := getItem(userID)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, nil
	}

	// create and send the response
	body, _ := json.Marshal(user)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: 200, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
