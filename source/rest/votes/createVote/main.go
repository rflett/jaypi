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
	var err error
	var status int

	authContext := services.GetAuthorizerContext(request.RequestContext)

	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	err = json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// create
	user := types.User{UserID: authContext.UserID}
	song := types.Song{
		SongID: reqBody.SongID,
		Name:   reqBody.Name,
		Album:  reqBody.Album,
		Artist: reqBody.Artist,
	}

	if status, err = user.AddVote(&song, reqBody.Position); err != nil {
		return services.ReturnError(err, status)
	}
	return services.ReturnNoContent()
}

func main() {
	lambda.Start(Handler)
}
