package types

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
	AuthProviderInternal    = "delegator"
	SNSPlatformGoogle       = "android"
	SNSPlatformApple        = "ios"
	UserAvatarDomain        = "assets.jaypi.com.au"
	GroupMembershipLimit    = 10
	VoteLimit               = 10
	AppEnvVar               = "APP_ENV"
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
	TestRequestContext = events.APIGatewayProxyRequestContext{
		Authorizer: map[string]interface{}{
			"AuthProvider":   TestAuthProvider,
			"AuthProviderId": TestAuthProviderId,
			"Name":           TestAuthProviderName,
			"UserID":         TestAuthProviderUserID,
		},
	}
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
