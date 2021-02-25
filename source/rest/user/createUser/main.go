package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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

	// TODO confirm user doesn't exist first

	// Create user password
	salt := generateSalt()
	saltStr := hex.EncodeToString(salt)
	hashedPassword, err := services.HashPassword(reqBody.Password, salt)
	if err != nil {
		logger.Log.Error().Err(err).Msg(fmt.Sprintf("Failed to create a user because a password hash failed"))
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
	}

	// create
	u := types.User{
		Name:     reqBody.Name,
		Email:    reqBody.Email,
		NickName: &reqBody.NickName,
		Password: &hashedPassword,
		Salt:     &saltStr,
	}
	createStatus, createErr := u.Create()
	if createErr != nil {
		return events.APIGatewayProxyResponse{Body: createErr.Error(), StatusCode: createStatus}, nil
	}

	// response
	responseBody, _ := json.Marshal(u)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(responseBody), StatusCode: http.StatusCreated, Headers: headers}, nil
}

// Generates a secure salt for the user
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
