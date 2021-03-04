package main

import (
	"encoding/json"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// requestBody is the expected body of the request
type requestBody struct {
	SongID   string `json:"songID"`
	Name     string `json:"name"`
	Album    string `json:"album"`
	Artist   string `json:"artist"`
	Position int    `json:"position"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &reqBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// create
	u := types.User{UserID: authContext.UserID}
	s := types.Song{
		SongID: reqBody.SongID,
		Name:   reqBody.Name,
		Album:  reqBody.Album,
		Artist: reqBody.Artist,
	}

	createStatus, createErr := u.AddVote(&s, reqBody.Position)
	if createErr != nil {
		return events.APIGatewayProxyResponse{Body: createErr.Error(), StatusCode: createStatus}, nil
	}

	// response
	return events.APIGatewayProxyResponse{Body: "", StatusCode: http.StatusNoContent}, nil
}

func main() {
	lambda.Start(Handler)
}
