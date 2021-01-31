package guess

// Guess is a song guess consisting of an artist and song name
type Guess struct {
	Artist string `json:"artist"`
	Song   string `json:"song"`
}

