package group

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dchest/uniuri"
	"github.com/google/uuid"
	logger "jjj.rflett.com/jjj-api/log"
	"net/http"
	"os"
	"time"
)

const (
	PrimaryKey = "GROUP"
	SortKey    = "#PROFILE"
)

var (
	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("ap-southeast-2")})
	db            = dynamodb.New(awsSession)
	table         = os.Getenv("JAYPI_TABLE")
)

// Group is way for users to be associated with each other
type Group struct {
	PK        string  `json:"-" dynamodbav:"PK"`
	SK        string  `json:"-" dynamodbav:"SK"`
	GroupID   string  `json:"groupID"`
	OwnerID   string  `json:"ownerID"`
	Name      string  `json:"name"`
	Code      string  `json:"code"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt *string `json:"updatedAt"`
}

// validateCode checks if a code already exists against a group and returns an error if it does
func validateCode(code string) error {
	return nil
}

// newCode creates a new code for the group
func (g *Group) newCode() {
	g.Code = uniuri.NewLen(6)
}

// Create the group and save it to the database
func (g *Group) Create() (status int, error error) {
	// set fields
	g.GroupID = uuid.NewString()
	g.PK = fmt.Sprintf("%s#%s", PrimaryKey, g.GroupID)
	g.SK = fmt.Sprintf("%s#%s", SortKey, g.GroupID)
	g.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	g.newCode()

	// create item
	av, _ := dynamodbattribute.MarshalMap(g)

	// create input
	input := &dynamodb.PutItemInput{
		TableName:    aws.String(table),
		Item:         av,
		ReturnValues: aws.String("NONE"),
	}

	// add to table
	_, err := db.PutItem(input)

	// handle errors
	if err != nil {
		logger.Log.Error().Err(err).Str("groupID", g.GroupID).Msg("Error adding group to table")
		return http.StatusInternalServerError, err
	}

	logger.Log.Info().Str("groupID", g.GroupID).Msg("Successfully added group to table")
	return http.StatusCreated, nil
}
