package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"testing"
)

func TestGetAvatarURL(t *testing.T) {
	request := events.APIGatewayProxyRequest{
		RequestContext: types.TestRequestContext,
	}

	response, err := Handler(request)
	assert.Nil(t, err)

	if assert.NotNil(t, response) {
		assert.Equal(t, http.StatusCreated, response.StatusCode, "Expected 201 Created status")
		assert.NotNil(t, response.Body)
	}
}