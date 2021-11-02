package types

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt"
	"os"
)

const (
	PartitionKey = "PK"
	SortKey      = "SK"

	GroupPartitionKey     = "GROUP"
	GroupSortKey          = "#PROFILE"
	GroupCodePartitionKey = "GROUP"
	GroupCodeSortKey      = "#CODE"
	GamePartitionKey      = "GROUP"
	GameSortKey           = "GAME"

	SongPartitionKey = "SONG"
	SongSortKey      = "#PROFILE"

	UserPartitionKey             = "USER"
	UserSortKey                  = "#PROFILE"
	UserAuthProviderPartitionKey = "USER"
	UserAuthProviderSortKey      = "#PROVIDER_ID"
	EndpointSortKey              = "#ENDPOINT"

	PlayCountPartitionKey   = "PLAYCOUNT"
	PlayCountSortKey        = "CURRENT"
	PlayedSongsPartitionKey = "PLAYEDSONGS"
	PlayedSongsSortKey      = "CURRENT"

	GSI = "GSI1"

	AuthProviderGoogle    = "google"
	AuthProviderGitHub    = "github"
	AuthProviderFacebook  = "facebook"
	AuthProviderInstagram = "instagram"
	AuthProviderSpotify   = "spotify"
	AuthProviderInternal  = "delegator"

	SNSPlatformGoogle = "android"
	SNSPlatformApple  = "ios"

	GroupMembershipLimit = 10
	VoteLimit            = 10
	AppEnvVar            = "APP_ENV"

	TestAuthProvider        = "delegator"
	TestAuthProviderId      = "ryan.flett1@gmail.com"
	TestAuthProviderName    = "Ryan"
	TestAuthProviderUserID  = "2e26e7dc-3f8c-456d-9d1b-8ce5b6447585"
	TestAuthProviderGroupID = "22abc6b1-3947-466d-8c62-6a73d82fb24e"
	TestAuthProviderPass    = ""
)

var (
	JWTSigningSecret   = "jaypi-private-key-staging"
	DynamoTable        = "jaypi-staging"
	AssetsBucket       = "jaypi-assets-staging"
	AssetsDomain       = "assets.staging.jaypi.online"
	TestRequestContext = events.APIGatewayProxyRequestContext{
		Authorizer: map[string]interface{}{
			"AuthProvider":   TestAuthProvider,
			"AuthProviderId": TestAuthProviderId,
			"Name":           TestAuthProviderName,
			"UserID":         TestAuthProviderUserID,
		},
	}
)

func init() {
	if v, ok := os.LookupEnv(AppEnvVar); ok {
		DynamoTable = fmt.Sprintf("jaypi-%s", v)
		JWTSigningSecret = fmt.Sprintf("jaypi-private-key-%s", v)
		AssetsBucket = fmt.Sprintf("jaypi-assets-%s", v)
		AssetsDomain = fmt.Sprintf("assets.%s.jaypi.online", v)
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

// PlayedSongs is a repr of the item that tracks the songs that have been played
type PlayedSongs struct {
	PK      string   `json:"-" dynamodbav:"PK"`
	SK      string   `json:"-" dynamodbav:"SK"`
	SongIDs []string `json:"songIDs"`
}
