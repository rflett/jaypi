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
)

type requestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// unmarshall request body to requestBody struct
	reqBody := requestBody{}
	err := json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	loginUser := types.User{
		Email:          reqBody.Email,
		AuthProvider:   aws.String(types.AuthProviderInternal),
		AuthProviderId: &reqBody.Email,
	}

	// confirm user exists
	exists, err := loginUser.Exists("AuthProviderId")
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}
	if !exists {
		return events.APIGatewayProxyResponse{Body: errors.New("Email or password are incorrect").Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// get the user
	getStatus, err := loginUser.GetByAuthProviderId()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: getStatus}, nil
	}

	// check password
	if !services.ComparePasswords(*loginUser.Password, reqBody.Password) {
		logger.Log.Warn().Str("userID", loginUser.UserID).Msg("Passwords don't match")
		return events.APIGatewayProxyResponse{Body: errors.New("password is incorrect").Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// create token
	token, err := loginUser.CreateToken()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusInternalServerError}, nil
	}

	// response
	loginResponse := types.LoginResponse{
		User:      loginUser,
		Token:     token,
		TokenType: "Bearer",
	}
	body, _ := json.Marshal(loginResponse)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: http.StatusOK, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
