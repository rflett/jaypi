package types

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/dgrijalva/jwt-go"
	"os"
)

const (
	PartitionKey                 = "PK"
	SortKey                      = "SK"
	GroupPartitionKey            = "GROUP"
	GroupSortKey                 = "#PROFILE"
	SongPartitionKey             = "SONG"
	SongSortKey                  = "#PROFILE"
	UserPartitionKey             = "USER"
	UserSortKey                  = "#PROFILE"
	GroupCodePartitionKey        = "GROUP"
	GroupCodeSortKey             = "#CODE"
	UserAuthProviderPartitionKey = "USER"
	UserAuthProviderSortKey      = "#PROVIDER_ID"
	PlayCountPartitionKey        = "PLAYCOUNT"
	PlayCountSortKey             = "CURRENT"
	GameSortKey                  = "GAME"
	EndpointSortKey              = "#ENDPOINT"
	GSI                          = "GSI1"
	AuthProviderGoogle           = "google"
	AuthProviderGitHub           = "github"
	AuthProviderFacebook         = "facebook"
	AuthProviderInstagram        = "instagram"
	AuthProviderSpotify          = "spotify"
	AuthProviderInternal         = "delegator"
	SNSPlatformGoogle            = "android"
	SNSPlatformApple             = "ios"
	UserAvatarDomain             = "assets.jaypi.com.au"
	GroupMembershipLimit         = 10
	VoteLimit                    = 10
	AppEnvVar                    = "APP_ENV"
	AssetsBucketVar              = "ASSETS_BUCKET"
	TestAuthProvider             = "delegator"
	TestAuthProviderId           = "ryan.flett1@gmail.com"
	TestAuthProviderName         = "Ryan"
	TestAuthProvierUserID        = "2e26e7dc-3f8c-456d-9d1b-8ce5b6447585"
	TestAuthProviderGroupID        = "22abc6b1-3947-466d-8c62-6a73d82fb24e"
	TestAuthProvierPass          = "Socom#3"
)

var (
	JWTSigningSecret   = "jaypi-private-key-staging"
	DynamoTable        = "jaypi-staging"
	AssetsBucket       = "jaypi-staging-rfacc"
	TestRequestContext = events.APIGatewayProxyRequestContext{
		Authorizer: map[string]interface{}{
			"AuthProvider":   TestAuthProvider,
			"AuthProviderId": TestAuthProviderId,
			"Name":           TestAuthProviderName,
			"UserID":         TestAuthProvierUserID,
		},
	}
)

func init() {
	if v, ok := os.LookupEnv(AppEnvVar); ok {
		DynamoTable = fmt.Sprintf("jaypi-%s", v)
		JWTSigningSecret = fmt.Sprintf("jaypi-private-key-%s", v)
	}
	if v, ok := os.LookupEnv(AssetsBucketVar); ok {
		AssetsBucket = v
	}
}

type PlayCount struct {
	Value *string `json:"value"`
}

type ChuneRefreshBody struct {
	SongID string `json:"songID"`
}

type BeanCounterBody struct {
	SongID string `json:"songID"`
}

type ScoreTakerBody struct {
	Points int    `json:"points"`
	UserID string `json:"userID"`
}

type CrierBody struct {
	UserID string `json:"userID"`
	Notification
}

type LoginResponse struct {
	User      User   `json:"user"`
	Token     string `json:"token"`
	TokenType string `json:"tokenType"`
}

// UserClaims are the custom claims that embedded into the JWT token for authentication
type UserClaims struct {
	Name           string  `json:"name"`
	Picture        *string `json:"picture"`
	AuthProvider   string  `json:"https://delegator.com.au/AuthProvider"`
	AuthProviderId string  `json:"https://delegator.com.au/AuthProviderId"`
	jwt.StandardClaims
}

// userAuthProvider represents a user and their AuthProviderId
type userAuthProvider struct {
	PK             string `json:"-" dynamodbav:"PK"`
	SK             string `json:"-" dynamodbav:"SK"`
	UserID         string `json:"userID"`
	AuthProviderId string `json:"authProviderId"`
	AuthProvider   string `json:"authProvider"`
}

// songVote is a votes in a users top 10
type songVote struct {
	PK     string `json:"-" dynamodbav:"PK"`
	SK     string `json:"-" dynamodbav:"SK"`
	SongID string `json:"songID"`
	UserID string `json:"userID"`
	Rank   int    `json:"rank"`
}

func (s *songVote) GetAsSong() (Song, error) {
	song := Song{
		SongID: s.SongID,
	}
	if err := song.Get(); err != nil {
		return Song{}, err
	}
	song.Rank = &s.Rank
	return song, nil
}
