package types

const (
	GroupPrimaryKey            = "GROUP"
	GroupSortKey               = "#PROFILE"
	SongPrimaryKey             = "SONG"
	SongSortKey                = "#PROFILE"
	UserPrimaryKey             = "USER"
	UserSortKey                = "#PROFILE"
	GroupCodePrimaryKey        = "GROUP"
	GroupCodeSortKey           = "#CODE"
	UserAuthProviderPrimaryKey = "USER"
	UserAuthProviderSortKey    = "#PROVIDER_ID"
	PlayCountPrimaryKey        = "PLAYCOUNT"
	PlayCountSortKey           = "CURRENT"
	GSI                        = "GSI1"
	AuthProviderGoogle         = "google"
	AuthProviderGitHub         = "github"
	AuthProviderFacebook       = "facebook"
	AuthProviderInternal       = "delegator"
	SNSPlatformGoogle          = "android"
	SNSPlatformApple           = "ios"
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

type LoginResponse struct {
	User      User   `json:"user"`
	Token     string `json:"token"`
	TokenType string `json:"tokenType"`
}

type AuthorizerContext struct {
	AuthProvider   string
	AuthProviderId string
	Name           string
	UserID         string
}
