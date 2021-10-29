package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"testing"
)

func TestGetUser(t *testing.T) {
	request := events.APIGatewayProxyRequest{
		RequestContext: types.TestRequestContext,
		PathParameters: map[string]string{
			"userId": types.TestAuthProviderUserID,
		},
		QueryStringParameters: map[string]string{
			"withVotes":  "true",
			"withGroups": "true",
		},
	}

	response, err := Handler(request)
	assert.Nil(t, err)

	if assert.NotNil(t, response) {
		getUserResponse := types.User{}
		err = json.Unmarshal([]byte(response.Body), &getUserResponse)
		assert.Nil(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode, "Expected 200 OK status")
		assert.GreaterOrEqual(t, len(*getUserResponse.Votes), 1)
		assert.GreaterOrEqual(t, len(*getUserResponse.Groups), 1)
	}
}
