package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"testing"
)

func TestSigninSuccess(t *testing.T) {
	bodyAsString, _ := json.Marshal(&RequestBody{
		Email:    types.TestAuthProviderId,
		Password: types.TestAuthProviderPass,
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

		assert.Equal(t, http.StatusOK, response.StatusCode, "Expected 200 OK status")
		assert.NotNil(t, loginResponse.Token)
		assert.NotNil(t, loginResponse.User)
		assert.Equal(t, "Bearer", loginResponse.TokenType, "TokenType should be Bearer")
	}
}
