package main

import (
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"strconv"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// get userId from pathParameters
	userID := request.PathParameters["userId"]

	// whether to get the users votes/groups as well
	withVotes, _ := strconv.ParseBool(request.QueryStringParameters["withVotes"])
	withGroups, _ := strconv.ParseBool(request.QueryStringParameters["withGroups"])

	// get user
	user := types.User{UserID: userID}
	if status, err := user.GetByUserID(); err != nil {
		return services.ReturnError(err, status)
	}

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

	// get their votes if required
	if withVotes {
		// get the members votes
		votes, voteErr := user.GetVotes()
		if voteErr == nil {
			user.Votes = &votes
		}
	}

	// get their groups if required
	if withGroups {
		// get the members votes
		groups, err := user.GetGroups()
		if err == nil {
			user.Groups = &groups
		}
	}

	// response
	return services.ReturnJSON(user, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
