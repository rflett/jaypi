package main

import (
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

type responseBody struct {
	Votes []types.Song `json:"votes"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get userId from pathParameters
	userID := request.PathParameters["userId"]

	// get user
	user := types.User{UserID: userID}

	// users can get themselves without doing the group check
	if authContext.UserID != userID {
		inSameGroup, err := services.UsersAreInSameGroup(authContext.UserID, userID)
		if err != nil {
			return services.ReturnError(err, http.StatusBadRequest)
		}
		if !inSameGroup {
			return services.ReturnError(errors.New("You have to a member of the group to do this"), http.StatusForbidden)
		}
	}

	// get their votes
	votes, voteErr := user.GetVotes()
	if voteErr == nil {
		user.Votes = &votes
	}

	// response
	rb := responseBody{Votes: votes}
	return services.ReturnJSON(rb, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
