package types

import (
	"github.com/dgrijalva/jwt-go"
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
)

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
