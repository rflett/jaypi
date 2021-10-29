package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// requestBody is the expected body of the request
type requestBody struct {
	Upsert []types.Song `json:"upsert"`
	Delete []string     `json:"delete"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var err error
	var status int

	// get the user
	authContext := services.GetAuthorizerContext(request.RequestContext)
	user := types.User{UserID: authContext.UserID}

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	err = json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// validate
	if len(reqBody.Upsert) > types.VoteLimit {
		return services.ReturnError(
			errors.New(fmt.Sprintf("A maximum of %d votes is allowed.", types.VoteLimit)), http.StatusBadRequest,
		)
	}

	// delete votes first
	for _, toDelete := range reqBody.Delete {
		status, err = user.RemoveVote(&toDelete)
		if err != nil {
			return services.ReturnError(err, status)
		}
	}

	// add other votes
	for _, toUpsert := range reqBody.Upsert {
		status, err = user.AddVote(&toUpsert)
		if err != nil {
			return services.ReturnError(err, status)
		}
	}

	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
