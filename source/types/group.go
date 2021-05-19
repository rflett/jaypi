package types

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dchest/uniuri"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"net/http"
	"time"
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

// GroupCode represents a group code used for inviting people
type GroupCode struct {
	PK      string `json:"-" dynamodbav:"PK"`
	SK      string `json:"-" dynamodbav:"SK"`
	GroupID string `json:"groupID"`
	Code    string `json:"code"`
}

// groupMember represents a users membership in a group
type groupMember struct {
	PK      string `json:"-" dynamodbav:"PK"`
	SK      string `json:"-" dynamodbav:"SK"`
	UserID  string `json:"userID"`
	GroupID string `json:"groupID"`
}

// Create the group and save it to the database
func (g *Group) Create() (status int, error error) {
	// set fields
	g.GroupID = uuid.NewString()
	g.PK = fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)
	g.SK = fmt.Sprintf("%s#%s", GroupSortKey, g.GroupID)
	g.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// create item
	av, _ := dynamodbattribute.MarshalMap(g)

	// create input
	input := &dynamodb.PutItemInput{
		TableName:    &clients.DynamoTable,
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}

	// add to table
	_, err := clients.DynamoClient.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("Error adding group to table")
		return http.StatusInternalServerError, err
	}

	// create code
	codeErr := g.NewCode()
	if codeErr != nil {
		return http.StatusInternalServerError, codeErr
	}

	// add the owner as a member
	joinStatus, joinErr := g.AddUser(g.OwnerID)
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

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#N":  aws.String("name"),
			"#UA": aws.String("updatedAt"),
			"#O":  aws.String("ownerID"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":ua": {
				S: g.UpdatedAt,
			},
			":n": {
				S: &g.Name,
			},
			":o": {
				S: &g.OwnerID,
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupSortKey, g.GroupID)),
			},
		},
		ReturnValues:        aws.String("NONE"),
		TableName:           &clients.DynamoTable,
		ConditionExpression: aws.String("#O = :o"),
		UpdateExpression:    aws.String("SET #N = :n, #UA = :ua"),
	}

	_, err := clients.DynamoClient.UpdateItem(input)

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
func (g *Group) Get() (status int, error error) {
	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupSortKey, g.GroupID)),
			},
		},
		TableName: &clients.DynamoTable,
	}

	// getItem
	result, err := clients.DynamoClient.GetItem(input)

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
			logger.Log.Error().Err(aerr).Str("groupID", g.GroupID).Msg("error getting group from table")
			return responseStatus, aerr
		} else {
			logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error getting group from table")
			return http.StatusInternalServerError, err
		}
	}

	if len(result.Item) == 0 {
		return http.StatusNotFound, nil
	}

	// unmarshal item into struct
	err = dynamodbattribute.UnmarshalMap(result.Item, &g)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("failed to unmarshal dynamo item to group")
	}

	// get the group code
	g.Code, _ = g.GetCode()
	return http.StatusOK, nil
}

func (g *Group) Delete() (status int, error error) {
	// delete the group owner membership
	owner := User{UserID: g.OwnerID}
	_, _ = owner.LeaveGroup(g.GroupID)

	// inputs
	deleteGroupCodeInput := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupCodeSortKey, g.Code)),
			},
		},
		TableName: &clients.DynamoTable,
	}
	deleteGroupInput := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)),
			},
			"SK": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupSortKey, g.GroupID)),
			},
		},
		TableName: &clients.DynamoTable,
	}

	// delete code from table
	if _, err := clients.DynamoClient.DeleteItem(deleteGroupCodeInput); err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error deleting group code item")
	}

	// delete group from table
	if _, err := clients.DynamoClient.DeleteItem(deleteGroupInput); err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error deleting group item")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("groupID", g.GroupID).Msg("succesfully deleted group")
	return http.StatusNoContent, nil
}

// AddUser a user to a group
func (g *Group) AddUser(userID string) (status int, err error) {
	user := User{UserID: userID}
	member := groupMember{
		PK:      fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID),
		SK:      fmt.Sprintf("%s#%s", UserPrimaryKey, userID),
		GroupID: g.GroupID,
		UserID:  userID,
	}

	// get the users groups
	var groups []Group
	groups, err = user.GetGroups()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// check membership limit
	if len(groups) == GroupMembershipLimit {
		return http.StatusBadRequest, errors.New(fmt.Sprintf(
			"Group limit reached - you can only be a member of up to %d groups.", GroupMembershipLimit),
		)
	}

	// check if they are already in the group
	for _, group := range groups {
		if group.GroupID == g.GroupID {
			return http.StatusConflict, errors.New("User is already a member of this group")
		}
	}

	// create the new group membership
	av, _ := dynamodbattribute.MarshalMap(member)
	putMemberInput := &dynamodb.PutItemInput{
		TableName:    &clients.DynamoTable,
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}
	_, err = clients.DynamoClient.PutItem(putMemberInput)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Str("userID", userID).Msg("Error adding user to group")
		return http.StatusInternalServerError, err
	}

	// return the group
	logger.Log.Info().Str("groupID", g.GroupID).Str("userID", userID).Msg("Successfully added user to group")
	return http.StatusOK, nil
}

// GetCode returns the code for a group
func (g *Group) GetCode() (string, error) {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)),
			},
			":sk": {
				S: aws.String("#CODE"),
			},
		},
		KeyConditionExpression: aws.String("PK = :pk and begins_with(SK, :sk)"),
		ProjectionExpression:   aws.String("code"),
		TableName:              &clients.DynamoTable,
	}

	// query
	result, err := clients.DynamoClient.Query(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupId", g.GroupID).Msg("error querying group for code")
		return "", err
	}

	// unmarshal groupID into the Group struct
	gc := GroupCode{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &gc)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupId", g.GroupID).Msg("error unmarshalling groupId to GroupCode")
		fmt.Printf("Failed to unmarshal Record, %v", err)
		return "", err
	}
	return gc.Code, nil
}

// GetQRCode returns the group code as a QR code
func (g *Group) GetQRCode() (string, error) {
	code, getCodeErr := g.GetCode()
	if getCodeErr != nil {
		return "", getCodeErr
	}

	qrCode, qrErr := qrcode.Encode(code, qrcode.Low, 256)
	if qrErr != nil {
		logger.Log.Error().Err(qrErr).Str("groupID", g.GroupID).Msg("Unable to generate QR code for group code")
		return "", qrErr
	}

	return base64.StdEncoding.EncodeToString(qrCode), nil
}

// NewCode generates a new group code
func (g *Group) NewCode() error {
	var code string

	// attempt to create the code
	for i := 1; i <= 5; i++ {
		codeAttempt := uniuri.NewLen(6)
		if ok := validateGroupCode(codeAttempt); ok == nil {
			code = codeAttempt
			break
		}
	}

	// return the error if we couldn't create the code
	if &code == nil {
		newCodeError := errors.New("unable to generate new code")
		logger.Log.Error().Err(newCodeError).Str("groupID", g.GroupID)
		return newCodeError
	}

	gc := GroupCode{
		PK:      fmt.Sprintf("%s#%s", GroupCodePrimaryKey, g.GroupID),
		SK:      fmt.Sprintf("%s#%s", GroupCodeSortKey, code),
		GroupID: g.GroupID,
		Code:    code,
	}

	// add the code to the table
	av, _ := dynamodbattribute.MarshalMap(gc)
	input := &dynamodb.PutItemInput{
		TableName:    &clients.DynamoTable,
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}
	_, err := clients.DynamoClient.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("Error adding code to table")
		return err
	}

	// success
	logger.Log.Info().Str("groupID", g.GroupID).Msg("Successfully added code to table")
	g.Code = code
	return nil
}

// GetMembers returns all the members of a group
func (g *Group) GetMembers(withVotes bool) ([]User, error) {
	// get the users in the group
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)),
			},
			":pk": {
				S: aws.String(fmt.Sprintf("%s#", UserPrimaryKey)),
			},
		},
		IndexName:              aws.String(GSI),
		KeyConditionExpression: aws.String("SK = :sk and begins_with(PK, :pk)"),
		ProjectionExpression:   aws.String("userID"),
		TableName:              &clients.DynamoTable,
	}

	groupMembers, err := clients.DynamoClient.Query(input)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error getting group members")
		return []User{}, err
	}

	var users []User = nil
	for _, member := range groupMembers.Items {
		user := User{}
		if err = dynamodbattribute.UnmarshalMap(member, &user); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to unmarshal group member to user")
			continue
		}
		if _, err = user.GetByUserID(); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to get user")
			continue
		}
		if withVotes {
			// get the members votes
			votes, voteErr := user.GetVotes()
			if voteErr == nil {
				user.Votes = &votes
			}
		}
		users = append(users, user)
	}
	return users, nil
}

// GetGames returns the games in a group
func (g *Group) GetGames() ([]Game, error) {
	// get the users in the group
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupPrimaryKey, g.GroupID)),
			},
			":sk": {
				S: aws.String("GAME#"),
			},
		},
		KeyConditionExpression: aws.String("PK = :pk and begins_with(SK, :sk)"),
		TableName:              &clients.DynamoTable,
	}

	groupsGames, err := clients.DynamoClient.Query(input)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error getting groups games")
		return []Game{}, err
	}

	//goland:noinspection GoPreferNilSlice
	games := []Game{}
	for _, groupGame := range groupsGames.Items {
		game := Game{}
		if err = dynamodbattribute.UnmarshalMap(groupGame, &game); err != nil {
			logger.Log.Error().Err(err).Msg("Unable to unmarshal groupGame to game")
			continue
		}
		games = append(games, game)
	}
	return games, nil
}

// ValidateCode checks if a code already exists against a group and returns an error if it does
func validateGroupCode(code string) error {
	// input
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sk": {
				S: aws.String(fmt.Sprintf("%s#%s", GroupCodeSortKey, code)),
			},
			":pk": {
				S: aws.String(fmt.Sprintf("%s#", GroupCodePrimaryKey)),
			},
		},
		IndexName:              aws.String(GSI),
		KeyConditionExpression: aws.String("SK = :sk and begins_with(PK, :pk)"),
		ProjectionExpression:   aws.String("code"),
		TableName:              &clients.DynamoTable,
	}

	// query
	result, err := clients.DynamoClient.Query(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("code", code).Msg("error validating code")
		return err
	}

	// code doesn't exist
	if len(result.Items) == 0 {
		logger.Log.Info().Str("code", code).Msg("code does not exist")
		return nil
	}

	// code exists
	logger.Log.Info().Str("code", code).Msg("code already exists")
	return errors.New("code already exists")
}
