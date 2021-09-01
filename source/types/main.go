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
	GameSortKey                = "GAME"
	EndpointSortKey            = "#ENDPOINT"
	GSI                        = "GSI1"
	AuthProviderGoogle         = "google"
	AuthProviderGitHub         = "github"
	AuthProviderFacebook       = "facebook"
	AuthProviderInstagram      = "instagram"
	AuthProviderSpotify        = "spotify"
	AuthProviderInternal       = "delegator"
	SNSPlatformGoogle          = "android"
	SNSPlatformApple           = "ios"
	UserAvatarDomain           = "assets.jaypi.com.au"
	GroupMembershipLimit       = 10
	VoteLimit				   = 10
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
