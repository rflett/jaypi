package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"jjj.rflett.com/jjj-api/types/jjj"
	"net/http"
	"os"
)

var (
	config = &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_SECRET_ID"),
		TokenURL:     spotify.TokenURL,
	}
	client              = spotify.Client{}
	songs  []types.Song = nil
)

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	query := request.QueryStringParameters["query"]

	// search spotify
	results, err := client.SearchOpt(query, spotify.SearchTypeTrack, &spotify.Options{
		Limit: aws.Int(3),
	})
	if err != nil {
		logger.Log.Error().Err(err).Str("track", query).Msg("Unable to search spotify for track")
		return services.ReturnError(err, http.StatusBadRequest)
	}

	// no search results
	if results.Tracks.Total == 0 {
		return services.ReturnJSON(songs, http.StatusOK)
	}

	for _, track := range results.Tracks.Tracks {
		// get the album artwork
		var artwork []jjj.ArtworkSize
		for _, art := range track.Album.Images {
			artwork = append(artwork, jjj.ArtworkSize{
				Url:    art.URL,
				Width:  art.Width,
				Height: art.Height,
			})
		}

		// create the song
		song := &types.Song{
			SongID:  track.SimpleTrack.ID.String(),
			Name:    track.SimpleTrack.Name,
			Album:   track.Album.Name,
			Artist:  track.Artists[0].Name,
			Artwork: &artwork,
		}
		songs = append(songs, *song)
	}

	return services.ReturnJSON(songs, http.StatusOK)
}

func init() {
	token, _ := config.Token(context.Background())
	client = spotify.Authenticator{}.NewClient(token)
}

func main() {
	lambda.Start(Handler)
}
