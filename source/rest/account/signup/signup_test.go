package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"testing"
)

func TestSignupSuccess(t *testing.T) {
	randStr := services.RandStringRunes(6)

	bodyAsString, _ := json.Marshal(&RequestBody{
		Name:     randStr,
		NickName: randStr,
		Email:    fmt.Sprintf("%s@gmail.com", randStr),
		Password: randStr,
	})

	request := events.APIGatewayProxyRequest{
		Body: string(bodyAsString),
	}

	response, err := Handler(request)
	assert.Nil(t, err)

	if assert.NotNil(t, response) {
		loginResponse := types.LoginResponse{}
		err = json.Unmarshal([]byte(response.Body), &loginResponse)
		assert.Nil(t, err)

		assert.Equal(t, http.StatusCreated, response.StatusCode, "Expected 201 Created status")
		assert.NotNil(t, loginResponse.Token)
		assert.NotNil(t, loginResponse.User)
		assert.Equal(t, "Bearer", loginResponse.TokenType, "TokenType should be Bearer")
	}
}
