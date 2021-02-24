package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"io/ioutil"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authCode := getAuthCode(request.PathParameters)
	if authCode == "" {
		return writeError(errors.New("MissingAuthCode"), "An authorisation code wasn't provided")
	}

	providerName := request.PathParameters["provider"]
	provider, err := services.GetOauthProvider(providerName)
	if err != nil {
		return writeError(err, "Failed to retrieve an oauth provider by name")
	}

	// Retrieve an auth token from the single use code
	userAuthToken, err := provider.Exchange(context.Background(), authCode)
	if err != nil {
		return writeError(err, "Couldn't retrieve a token for the user")
	}

	// We now have a token for the user, get their data
	userClient := provider.Client(context.Background(), userAuthToken)
	userEmailResp, _ := userClient.Get(provider.GetProfileRequestUrl(userAuthToken))
	responseMap, err := getResponseContent(userEmailResp)
	if err != nil {
		return writeError(err, "Failed to retrieve response from oauth profile request")
	}

	// Convert from provider-specific into generic data
	userInfo := provider.GetGenericResponseData(responseMap)

	// Log the user in and receive a JWT
	return registerOrLoginOauthUser(userInfo, providerName), nil
}

// Different providers return the code in a different format. Try them all
func getAuthCode(params map[string]string) string {
	authCode := params["code"]

	if authCode == "" {
		// Try access_token
		authCode = params["access_token"]
	}
	return authCode
}

// Logs and returns an error message to the user
func writeError(err error, msg string) (events.APIGatewayProxyResponse, error) {
	logger.Log.Error().Err(err).Msg(msg)
	return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusBadRequest}, nil
}

// Reads the contents of the HTTP response and convert it into a map if possible
func getResponseContent(response *http.Response) (map[string]interface{}, error) {
	defer response.Body.Close()

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	responseMap := make(map[string]interface{})
	err = json.Unmarshal(responseBytes, &responseMap)
	if err != nil {
		return nil, err
	}

	return responseMap, nil
}

// Checks if an oauth user is already in the database, if not register them.
// Either way generate a JWT for the user that's specific to our application
func registerOrLoginOauthUser(userInfo types.OauthResponse, providerName string) events.APIGatewayProxyResponse {

	newUser := types.User{
		Name:           userInfo.Name,
		Email:          userInfo.Email,
		AuthProvider:   &providerName,
		AuthProviderId: &userInfo.Id,
		AvatarUrl:      &userInfo.Picture,
	}

	var err error
	var createStatus int
	if newUser.AlreadySignedUp() {
		// Log user in if the provider matches
		createStatus, err = newUser.Create()
	} else {
		// Log user in if the provider matches
		createStatus, err = newUser.Get()
	}
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: createStatus}
	}

	// response
	responseBody, _ := json.Marshal(newUser)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(responseBody), StatusCode: http.StatusCreated, Headers: headers}
}

func main() {
	lambda.Start(Handler)
}
