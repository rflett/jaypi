package types

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
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

func (g *Group) PKVal() string {
	return fmt.Sprintf("%s#%s", GroupPartitionKey, g.GroupID)
}

func (g *Group) SKVal() string {
	return fmt.Sprintf("%s#%s", GroupSortKey, g.GroupID)
}

// Create the group and save it to the database
func (g *Group) Create() (status int, error error) {
	// set fields
	g.GroupID = uuid.NewString()
	g.PK = g.PKVal()
	g.SK = g.SKVal()
	g.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// create item
	av, _ := attributevalue.MarshalMap(g)

	// create input
	input := &dynamodb.PutItemInput{
		TableName:    &DynamoTable,
		Item:         av,
		ReturnValues: dbTypes.ReturnValueNone,
	}

	// add to table
	_, err := clients.DynamoClient.PutItem(context.TODO(), input)

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
		ExpressionAttributeNames: map[string]string{
			"#N":  "name",
			"#UA": "updatedAt",
			"#O":  "ownerID",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":ua": &dbTypes.AttributeValueMemberS{Value: *g.UpdatedAt},
			":n":  &dbTypes.AttributeValueMemberS{Value: g.Name},
			":o":  &dbTypes.AttributeValueMemberS{Value: g.OwnerID},
		},
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: g.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: g.SKVal()},
		},
		ReturnValues:        dbTypes.ReturnValueNone,
		TableName:           &DynamoTable,
		ConditionExpression: aws.String("#O = :o"),
		UpdateExpression:    aws.String("SET #N = :n, #UA = :ua"),
	}

	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error updating group")
		return http.StatusInternalServerError, err
	}

	return http.StatusNoContent, nil
}

// NominateOwner sets a new owner of the group
func (g *Group) NominateOwner(userID string) (status int, error error) {
	// set fields
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	g.UpdatedAt = &updatedAt

	// update query
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]string{
			"#UA": "updatedAt",
			"#O":  "ownerID",
		},
		ExpressionAttributeValues: map[string]dbTypes.AttributeValue{
			":ua": &dbTypes.AttributeValueMemberS{Value: *g.UpdatedAt},
			":o":  &dbTypes.AttributeValueMemberS{Value: userID},
		},
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: g.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: g.SKVal()},
		},
		ReturnValues:     dbTypes.ReturnValueNone,
		TableName:        &DynamoTable,
		UpdateExpression: aws.String("SET #O = :o, #UA = :ua"),
	}

	_, err := clients.DynamoClient.UpdateItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error updating group owner")
		return http.StatusInternalServerError, err
	}

	return http.StatusNoContent, nil
}

// Get the group from the table
func (g *Group) Get() (status int, error error) {
	// get query
	input := &dynamodb.GetItemInput{
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: g.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: g.SKVal()},
		},
		TableName: &DynamoTable,
	}

	// getItem
	result, err := clients.DynamoClient.GetItem(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error getting group from table")
		return http.StatusInternalServerError, err
	}

	if len(result.Item) == 0 {
		return http.StatusNotFound, nil
	}

	// unmarshal item into struct
	err = attributevalue.UnmarshalMap(result.Item, &g)
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
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: g.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: fmt.Sprintf("%s#%s", GroupCodeSortKey, g.Code)},
		},
		TableName: &DynamoTable,
	}
	deleteGroupInput := &dynamodb.DeleteItemInput{
		Key: map[string]dbTypes.AttributeValue{
			PartitionKey: &dbTypes.AttributeValueMemberS{Value: g.PKVal()},
			SortKey:      &dbTypes.AttributeValueMemberS{Value: g.SKVal()},
		},
		TableName: &DynamoTable,
	}

	// delete code from table
	if _, err := clients.DynamoClient.DeleteItem(context.TODO(), deleteGroupCodeInput); err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error deleting group code item")
	}

	// delete group from table
	if _, err := clients.DynamoClient.DeleteItem(context.TODO(), deleteGroupInput); err != nil {
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
		PK:      g.PKVal(),
		SK:      fmt.Sprintf("%s#%s", UserPartitionKey, userID),
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
	av, _ := attributevalue.MarshalMap(member)
	putMemberInput := &dynamodb.PutItemInput{
		TableName:    &DynamoTable,
		Item:         av,
		ReturnValues: dbTypes.ReturnValueNone,
	}
	_, err = clients.DynamoClient.PutItem(context.TODO(), putMemberInput)
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
	pkCondition := expression.Key(PartitionKey).Equal(expression.Value(g.PKVal()))
	skCondition := expression.Key(SortKey).BeginsWith(fmt.Sprintf("%s#", GroupCodeSortKey))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("code"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for GetCode func")
	}

	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	// query
	result, err := clients.DynamoClient.Query(context.TODO(), input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupId", g.GroupID).Msg("error querying group for code")
		return "", err
	}

	// unmarshal groupID into the Group struct
	gc := GroupCode{}
	err = attributevalue.UnmarshalMap(result.Items[0], &gc)
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
		PK:      g.PKVal(),
		SK:      fmt.Sprintf("%s#%s", GroupCodeSortKey, code),
		GroupID: g.GroupID,
		Code:    code,
	}

	// add the code to the table
	av, _ := attributevalue.MarshalMap(gc)
	input := &dynamodb.PutItemInput{
		TableName:    &DynamoTable,
		Item:         av,
		ReturnValues: dbTypes.ReturnValueNone,
	}
	_, err := clients.DynamoClient.PutItem(context.TODO(), input)

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
	pkCondition := expression.Key(PartitionKey).Equal(expression.Value(g.PKVal()))
	skCondition := expression.Key(SortKey).BeginsWith(fmt.Sprintf("%s#", UserPartitionKey))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("userID"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for GetMembers func")
	}

	// input
	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	groupMembers, err := clients.DynamoClient.Query(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error getting group members")
		return []User{}, err
	}

	var users []User = nil
	for _, member := range groupMembers.Items {
		user := User{}
		if err = attributevalue.UnmarshalMap(member, &user); err != nil {
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
	pkCondition := expression.Key(PartitionKey).Equal(expression.Value(g.PKVal()))
	skCondition := expression.Key(SortKey).BeginsWith(fmt.Sprintf("%s#", GameSortKey))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for GetGames func")
	}

	// input
	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeValues: expr.Values(),
	}

	groupsGames, err := clients.DynamoClient.Query(context.TODO(), input)
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("error getting groups games")
		return []Game{}, err
	}

	//goland:noinspection GoPreferNilSlice
	games := []Game{}
	for _, groupGame := range groupsGames.Items {
		game := Game{}
		if err = attributevalue.UnmarshalMap(groupGame, &game); err != nil {
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
	pkCondition := expression.Key(PartitionKey).BeginsWith(fmt.Sprintf("%s#", GroupCodePartitionKey))
	skCondition := expression.Key(SortKey).Equal(expression.Value(fmt.Sprintf("%s#%s", GroupCodeSortKey, code)))
	keyCondition := expression.KeyAnd(pkCondition, skCondition)

	projExpr := expression.NamesList(expression.Name("code"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCondition).WithProjection(projExpr).Build()

	if err != nil {
		logger.Log.Error().Err(err).Msg("error building expression for validateGroupCode func")
	}

	// input
	input := &dynamodb.QueryInput{
		TableName:                 &DynamoTable,
		IndexName:                 aws.String(GSI),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	}

	// query
	result, err := clients.DynamoClient.Query(context.TODO(), input)

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
