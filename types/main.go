package types

const (
	GroupPrimaryKey     = "GROUP"
	GroupSortKey        = "#PROFILE"
	SongPrimaryKey      = "SONG"
	SongSortKey         = "#PROFILE"
	UserPrimaryKey      = "USER"
	UserSortKey         = "#PROFILE"
	GroupCodePrimaryKey = "GROUP"
	GroupCodeSortKey    = "#CODE"
	PlayCountPrimaryKey = "PLAYCOUNT"
	PlayCountSortKey    = "CURRENT"
	GSI                 = "GSI1"
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
