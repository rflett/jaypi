package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"testing"
)

func TestCreateGame(t *testing.T) {
	name := services.RandStringRunes(12)
	desc := services.RandStringRunes(30)

	bodyAsString, _ := json.Marshal(&RequestBody{
		Name:        name,
		Description: desc,
	})

	request := events.APIGatewayProxyRequest{
		RequestContext: types.TestRequestContext,
		Body:           string(bodyAsString),
		PathParameters: map[string]string{
			"groupId": types.TestAuthProviderGroupID,
		},
	}

	response, err := Handler(request)
	assert.Nil(t, err)

	if assert.NotNil(t, response) {
		rb := types.Game{}
		err = json.Unmarshal([]byte(response.Body), &rb)
		assert.Nil(t, err)

		assert.Equal(t, http.StatusCreated, response.StatusCode, "Expected 201 Created status")
		assert.Equal(t, types.TestAuthProviderGroupID, rb.GroupID)
		assert.NotNil(t, name, rb.Name)
		assert.Equal(t, desc, rb.Description)
	}
}
