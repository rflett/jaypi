package main

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"net/http"
	"time"
)

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
	input := &s3.GetObjectInput{
		Bucket: &types.AssetsBucket,
		Key:    &avatarUuid,
	}
	psClient := s3.NewPresignClient(clients.S3Client, func(options *s3.PresignOptions) {
		options.Expires = 15 * time.Minute
	})

	presignResponse, err := psClient.PresignGetObject(context.TODO(), input)
	if err != nil {
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// response
	return services.ReturnJSON(presignResponse.URL, http.StatusCreated)
}

func main() {
	lambda.Start(Handler)
}
