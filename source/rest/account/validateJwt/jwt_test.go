package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestValidateJWT(t *testing.T) {
	request := events.APIGatewayProxyRequest{
		Body: "",
	}

	response, err := Handler(request)
	assert.Nil(t, err)

	if assert.NotNil(t, response) {
		assert.Equal(t, http.StatusNoContent, response.StatusCode, "Expected 204 No Content status")
	}
}
