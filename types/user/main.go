package user

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	logger "jjj.rflett.com/jjj-api/log"
	"jjj.rflett.com/jjj-api/types/song"
	"net/http"
	"os"
	"time"
)

const (
	PrimaryKey = "USER"
	SortKey    = "#PROFILE"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	table         = os.Getenv("JAYPI_TABLE")
)

// User is a User of the application
type User struct {
	PK        string  `json:"-" dynamodbav:"PK"`
	SK        string  `json:"-" dynamodbav:"SK"`
	UserID    string  `json:"userID"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	NickName  string  `json:"nickName"`
	Email     *string `json:"email"`
	Points    int     `json:"points"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt *string `json:"updatedAt"`
}

// songVote is a votes in a users top 10
type songVote struct {
	PK       string `json:"-" dynamodbav:"PK"`
	SK       string `json:"-" dynamodbav:"SK"`
	SongID   string `json:"songID"`
	Position int    `json:"position"`
}

// voteCount returns the number of votes a user already has
func (u *User) voteCount() (count int, error error) {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, u.UserID)),
			},
			":sk": {
				S: aws.String(fmt.Sprintf("%s#", "SONG")),
			},
		},
		KeyConditionExpression: aws.String("PK = :pk and begins_with(SK, :sk)"),
		ProjectionExpression:   aws.String("userID"),
		TableName:              &table,
	}

	queryResult, queryErr := db.Query(input)
	if queryErr != nil {
		logger.Log.Error().Err(queryErr).Str("userId", u.UserID).Msg("error getting user song count")
		return 0, queryErr
	}
	return int(*queryResult.Count), nil
}

// Create the user and save them to the database
func (u *User) Create() (status int, error error) {
	// set fields
	u.UserID = uuid.NewString()
	u.PK = fmt.Sprintf("%s#%s", PrimaryKey, u.UserID)
	u.SK = fmt.Sprintf("%s#%s", SortKey, u.UserID)
	u.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// create item
	av, _ := dynamodbattribute.MarshalMap(u)

	// create input
	input := &dynamodb.PutItemInput{
		TableName:    &table,
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}

	// add to table
	_, err := db.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("Error adding user to table")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("userID", u.UserID).Msg("Successfully added user to table")
	return http.StatusCreated, nil
}

// Update the user's attributes
func (u *User) Update() (status int, error error) {
	// set fields
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	u.UpdatedAt = &updatedAt

	pk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, u.UserID)),
	}
	sk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", SortKey, u.UserID)),
	}

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#NN": aws.String("nickName"),
			"#UA": aws.String("updatedAt"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": &pk,
			":sk": &sk,
			":nn": {
				S: aws.String(u.NickName),
			},
			":ua": {
				S: aws.String(*u.UpdatedAt),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": &pk,
			"SK": &sk,
		},
		ReturnValues:        aws.String("NONE"),
		TableName:           &table,
		ConditionExpression: aws.String("PK = :pk and SK = :sk"),
		UpdateExpression:    aws.String("SET #NN = :nn, #UA = :ua"),
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
			logger.Log.Error().Err(aerr).Str("userID", u.UserID).Msg("error updating user")
			return responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("userID", u.UserID).Msg("error updating user")
			return http.StatusInternalServerError, err
		}
	}

	return http.StatusNoContent, nil
}

// AddVote adds a song as a votes for the user
func (u *User) AddVote(s *song.Song, position int) (status int, error error) {
	// check if song exists and add it if it doesn't
	exists, existsErr := s.Exists()
	if existsErr != nil {
		return http.StatusInternalServerError, existsErr
	}

	if !exists {
		createSongErr := s.Create()
		if createSongErr != nil {
			return http.StatusInternalServerError, createSongErr
		}
	}

	// don't allow more than 10 votes
	vc, vcErr := u.voteCount()
	if vcErr != nil {
		return http.StatusInternalServerError, vcErr
	}
	if vc >= 10 {
		tooManyCountsErr := errors.New("user already has 10 song votes")
		logger.Log.Error().Err(tooManyCountsErr).Str("userID", u.UserID).Msg("User has maxxed out their votes")
		return http.StatusBadRequest, tooManyCountsErr
	}

	// set fields
	sv := songVote{
		PK:       fmt.Sprintf("%s#%s", PrimaryKey, u.UserID),
		SK:       fmt.Sprintf("%s#%s", "SONG", s.SongID),
		SongID:   s.SongID,
		Position: position,
	}

	// create item
	av, _ := dynamodbattribute.MarshalMap(sv)
	input := &dynamodb.PutItemInput{
		Item:         av,
		ReturnValues: aws.String("NONE"),
		TableName:    &table,
	}

	// add to table
	_, err := db.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", u.UserID).Str("songID", s.SongID).Msg("Error adding votes")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("userID", u.UserID).Str("songID", s.SongID).Msg("Added user votes")
	return http.StatusNoContent, nil
}

// RemoveVote removes a song as a users vote
func (u *User) RemoveVote(songID *string) (status int, error error) {
	// set fields
	pk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, u.UserID)),
	}
	sk := dynamodb.AttributeValue{
		S: aws.String(fmt.Sprintf("%s#%s", "SONG", *songID)),
	}

	// delete query
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": &pk,
			"SK": &sk,
		},
		TableName: &table,
	}

	// delete from table
	_, err := db.DeleteItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("songID", *songID).Str("userID", u.UserID).Msg("Error removing song from user")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("songID", *songID).Str("userID", u.UserID).Msg("Removed song vote from user")
	return http.StatusNoContent, nil
}

// Get the user from the table
func Get(userID string) (user User, status int, error error) {
	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", PrimaryKey, userID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", SortKey, userID)),
			},
		},
		TableName: &table,
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
			logger.Log.Error().Err(aerr).Str("userID", userID).Msg("error getting user from table")
			return User{}, responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("userID", userID).Msg("error getting user from table")
			return User{}, http.StatusInternalServerError, err
		}
	}

	if len(result.Item) == 0 {
		return User{}, http.StatusNotFound, nil
	}

	// unmarshal item into struct
	err = dynamodbattribute.UnmarshalMap(result.Item, &user)
	if err != nil {
		logger.Log.Error().Err(err).Str("userID", userID).Msg("failed to unmarshal dynamo item to user")
	}

	return user, http.StatusOK, nil
}
