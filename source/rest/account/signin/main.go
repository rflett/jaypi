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

type RequestBody struct {
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

	email := strings.ToLower(reqBody.Email)
	loginUser := types.User{
		Email:          email,
		AuthProvider:   aws.String(types.AuthProviderInternal),
		AuthProviderId: &email,
	}

	// confirm user exists
	exists, err := loginUser.Exists("AuthProviderId")
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}
	if !exists {
		return services.ReturnError(errors.New("Email or password are incorrect"), http.StatusBadRequest)
	}

	// get the user
	status, err := loginUser.GetByAuthProviderId()
	if err != nil {
		return services.ReturnError(err, status)
	}

	// check password
	if !services.ComparePasswords(*loginUser.Password, reqBody.Password) {
		logger.Log.Warn().Str("userID", loginUser.UserID).Msg("Passwords don't match")
		return services.ReturnError(errors.New("Email or password are incorrect"), http.StatusBadRequest)
	}

	// create token
	token, err := loginUser.CreateToken()
	if err != nil {
		return services.ReturnError(err, http.StatusInternalServerError)
	}

	// get their groups
	groups, err := loginUser.GetGroups()
	if err == nil {
		loginUser.Groups = &groups
	}

	// response
	loginResponse := types.LoginResponse{
		User:      loginUser,
		Token:     token,
		TokenType: "Bearer",
	}
	return services.ReturnJSON(loginResponse, http.StatusOK)
}

func main() {
	lambda.Start(Handler)
}
