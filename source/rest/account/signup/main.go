package main

import (
	"encoding/json"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"strings"
)

// RequestBody is the expected body of the request
type RequestBody struct {
	Name     string `json:"name"`
	NickName string `json:"nickName"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// unmarshall request body to RequestBody struct
	reqBody := RequestBody{}
	err := json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	newUser := types.User{
		Name:           reqBody.Name,
		Email:          strings.ToLower(reqBody.Email),
		NickName:       &reqBody.NickName,
		AuthProvider:   aws.String(types.AuthProviderInternal),
		AuthProviderId: &reqBody.Email,
	}

	// confirm user doesn't exist first
	exists, err := newUser.Exists("AuthProviderId")
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}
	if exists {
		return services.ReturnError(errors.New("That email is already in use"), http.StatusBadRequest)
	}

	// Create user password
	password, err := services.HashAndSaltPassword(reqBody.Password)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Failed to create a user because a password hash failed")
		return services.ReturnError(err, http.StatusBadRequest)
	}
	newUser.Password = &password

	// create the user
	status, err := newUser.Create()
	if err != nil {
		return services.ReturnError(err, status)
	}

	// create token
	token, err := newUser.CreateToken()
	if err != nil {
		return services.ReturnError(err, http.StatusInternalServerError)
	}

	// response
	loginResponse := types.LoginResponse{
		User:      newUser,
		Token:     token,
		TokenType: "Bearer",
	}
	return services.ReturnJSON(loginResponse, http.StatusCreated)
}

func main() {
	lambda.Start(Handler)
}
