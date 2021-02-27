package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/uuid"
	"io"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

// requestBody is the expected body of the request
type requestBody struct {
	Name     string `json:"name"`
	NickName string `json:"nickName"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	jsonErr := json.Unmarshal([]byte(request.Body), &reqBody)
	if jsonErr != nil {
		return events.APIGatewayProxyResponse{Body: jsonErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	newUser := types.User{
		Name:           reqBody.Name,
		Email:          reqBody.Email,
		NickName:       &reqBody.NickName,
		AuthProvider:   aws.String(types.AuthProviderInternal),
		AuthProviderId: &reqBody.Email,
	}

	// confirm user doesn't exist first
	exists, existsErr := newUser.Exists("AuthProviderId")
	if existsErr != nil {
		return events.APIGatewayProxyResponse{Body: existsErr.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	if exists {
		return events.APIGatewayProxyResponse{Body: errors.New("user already exists").Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// Create user password
	password, err := services.HashAndSaltPassword(reqBody.Password)
	if err != nil {
		logger.Log.Error().Err(err).Msg(fmt.Sprintf("Failed to create a user because a password hash failed"))
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}
	newUser.Password = &password

	// create the user
	createStatus, createErr := newUser.Create()
	if createErr != nil {
		return events.APIGatewayProxyResponse{Body: createErr.Error(), StatusCode: createStatus}, nil
	}

	// create token
	token, tokenErr := newUser.CreateToken()
	if tokenErr != nil {
		return events.APIGatewayProxyResponse{Body: tokenErr.Error(), StatusCode: http.StatusInternalServerError}, nil
	}

	// response
	loginResponse := types.LoginResponse{
		User:      newUser,
		Token:     token,
		TokenType: "Bearer",
	}
	body, _ := json.Marshal(loginResponse)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: http.StatusCreated, Headers: headers}, nil
}

// generateSalt creates a secure salt for the user
func generateSalt() []byte {
	saltBytes := make([]byte, 32)

	_, err := io.ReadFull(rand.Reader, saltBytes)
	if err != nil {
		// Backup generation
		saltBytes, _ = uuid.New().MarshalBinary()
	}

	return saltBytes
}

func main() {
	lambda.Start(Handler)
}
