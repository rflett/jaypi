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

func TestUpdateUser(t *testing.T) {
	bodyAsString, _ := json.Marshal(&RequestBody{
		NickName: services.RandStringRunes(6),
	})

	response, err := Handler(events.APIGatewayProxyRequest{
		RequestContext: types.TestRequestContext,
		Body:           string(bodyAsString),
	})
	assert.Nil(t, err)

	if assert.Nil(t, err) {
		assert.NotNil(t, response)
		assert.Equal(t, http.StatusNoContent, response.StatusCode, "Expected 204 No Content status")
	}
}
