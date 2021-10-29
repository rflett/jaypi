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

func TestCreateGroup(t *testing.T) {
	name := services.RandStringRunes(12)

	bodyAsString, _ := json.Marshal(&RequestBody{
		Name: name,
	})

	request := events.APIGatewayProxyRequest{
		RequestContext: types.TestRequestContext,
		Body:           string(bodyAsString),
	}

	response, err := Handler(request)
	assert.Nil(t, err)

	if assert.NotNil(t, response) {
		rb := types.Group{}
		err = json.Unmarshal([]byte(response.Body), &rb)
		assert.Nil(t, err)

		assert.Equal(t, http.StatusCreated, response.StatusCode, "Expected 201 Created status")
		assert.NotNil(t, name, rb.Name)
	}
}
