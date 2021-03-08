package main

import (
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"os"
	"time"
)

var (
	bucket = os.Getenv("ASSETS_BUCKET")
)

type responseBody struct {
	Url string `json:"url"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	authContext := services.GetAuthorizerContext(request.RequestContext)

	// only internal users can upload custom avatars
	if authContext.AuthProvider != types.AuthProviderInternal {
		return services.ReturnError(errors.New("Cannot upload custom avatar when using social login"), http.StatusBadRequest)
	}

	// generate a new avatar url
	user := types.User{UserID: authContext.UserID}
	avatarUuid, err := user.GenerateAvatarUrl()
	if err != nil {
		return services.ReturnError(err, http.StatusInternalServerError)
	}

	// get the pre-sign url
	req, _ := clients.S3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &avatarUuid,
	})

	// pre-sign the url for 15 minutes
	urlStr, err := req.Presign(15 * time.Minute)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// response
	body := responseBody{Url: urlStr}
	return services.ReturnJSON(body, http.StatusCreated)
}

func main() {
	lambda.Start(Handler)
}
