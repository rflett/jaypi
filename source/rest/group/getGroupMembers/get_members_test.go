package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"testing"
)

func TestGetGroupMembers(t *testing.T) {
	request := events.APIGatewayProxyRequest{
		RequestContext: types.TestRequestContext,
		PathParameters: map[string]string{
			"groupId": types.TestAuthProviderGroupID,
		},
	}

	response, err := Handler(request)
	assert.Nil(t, err)

	if assert.NotNil(t, response) {
		rb := ResponseBody{}
		err = json.Unmarshal([]byte(response.Body), &rb)
		assert.Nil(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode, "Expected 200 OK status")
		assert.GreaterOrEqual(t, len(rb.Members), 1)
	}
}
