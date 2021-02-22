package jjj

type Link struct {
	Entity         string  `json:"entity"`
	Arid           string  `json:"arid"`
	Url            string  `json:"url"`
	IdComponent    string  `json:"id_component"`
	Title          string  `json:"title"`
	MiniSynopsis   *string `json:"mini_synopsis"`
	ShortSynopsis  *string `json:"short_synopsis"`
	MediumSynopsis *string `json:"medium_synopsis"`
	Type           string  `json:"type"`
	Provider       string  `json:"provider"`
	External       bool    `json:"external"`
}

type ArtworkSize struct {
	Url         string `json:"url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	AspectRatio string `json:"aspect_ratio"`
}

type Artwork struct {
	Entity         string        `json:"entity"`
	Arid           string        `json:"arid"`
	Url            string        `json:"url"`
	Type           string        `json:"type"`
	Title          *string       `json:"title"`
	MiniSynopsis   *string       `json:"mini_synopsis"`
	ShortSynopsis  *string       `json:"short_synopsis"`
	MediumSynopsis *string       `json:"medium_synopsis"`
	Width          int           `json:"width"`
	Height         int           `json:"height"`
	Sizes          []ArtworkSize `json:"sizes"`
}

type Artist struct {
	Entity       string    `json:"entity"`
	Arid         string    `json:"arid"`
	Name         string    `json:"name"`
	Artwork      []Artwork `json:"artwork"`
	Links        []Link    `json:"links"`
	IsAustralian *bool     `json:"is_australian"`
	Type         string    `json:"type"`
	Role         *string   `json:"role"`
}

type Release struct {
	Entity         string    `json:"entity"`
	Arid           string    `json:"arid"`
	Title          string    `json:"title"`
	Format         string    `json:"format"`
	RecordLabel    *string   `json:"record_label"`
	ReleaseYear    string    `json:"release_year"`
	ReleaseAlbumID string    `json:"release_album_id"`
	Artwork        []Artwork `json:"artwork"`
	Links          []Link    `json:"links"`
	Artists        []Artist  `json:"artists"`
}

type Recording struct {
	Entity      string    `json:"entity"`
	Arid        string    `json:"arid"`
	Title       string    `json:"title"`
	Metadata    *string   `json:"metadata"`
	Description *string   `json:"description"`
	Duration    int       `json:"duration"`
	Artists     []Artist  `json:"artists"`
	Releases    []Release `json:"releases"`
	Artwork     []Artwork `json:"artwork"`
	Links       []Link    `json:"links"`
}

type Play struct {
	Entity     string    `json:"entity"`
	Arid       string    `json:"arid"`
	PlayedTime string    `json:"played_time"`
	ServiceID  string    `json:"service_id"`
	Recording  Recording `json:"recording"`
	Release    *Release  `json:"release"`
}

type ResponseBody struct {
	LastUpdated string `json:"last_updated"`
	NextUpdated string `json:"next_updated"`
	Next        *Play  `json:"next"`
	Now         *Play  `json:"now"`
	Prev        *Play  `json:"prev"`
}
