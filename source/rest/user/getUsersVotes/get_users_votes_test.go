package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"testing"
)

func TestGetUsersVotes(t *testing.T) {
	request := events.APIGatewayProxyRequest{
		RequestContext: types.TestRequestContext,
		PathParameters: map[string]string{
			"userId": "2e26e7dc-3f8c-456d-9d1b-8ce5b6447585",
		},
	}

	response, err := Handler(request)
	assert.Nil(t, err)

	if assert.NotNil(t, response) {
		rb := ResponseBody{}
		err = json.Unmarshal([]byte(response.Body), &rb)
		assert.Nil(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode, "Expected 200 OK status")
		assert.GreaterOrEqual(t, len(rb.Votes), 1)
	}
}
