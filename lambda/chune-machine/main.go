package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
	"io/ioutil"
	"jjj.rflett.com/jjj-api/clients"
	logger "jjj.rflett.com/jjj-api/log"
	"jjj.rflett.com/jjj-api/types"
	"jjj.rflett.com/jjj-api/types/jjj"
	"net/http"
	"os"
	"time"
)

const location = "Australia/Sydney"

var (
	chuneRefreshQueue = os.Getenv("REFRESH_QUEUE")
	beanCounterQueue  = os.Getenv("COUNTER_QUEUE")
)

// queueForCounter puts the songID on a queue to trigger the counter lambdas
func queueForCounter(songID *string) error {
	// create message body
	mb, _ := json.Marshal(types.BeanCounterBody{SongID: *songID})
	input := &sqs.SendMessageInput{
		DelaySeconds: aws.Int64(0),
		MessageBody:  aws.String(string(mb)),
		QueueUrl:     aws.String(beanCounterQueue),
	}

	// send the message to the queue
	message, err := clients.SQSClient.SendMessage(input)

	if err != nil {
		logger.Log.Error().Err(err).Str("songID", *songID).Msg("Unable to put the song onto the beanCounterQueue")
		return err
	}

	logger.Log.Info().Str("songID", *songID).Str("messageID", *message.MessageId).Msg("Successfully put the song onto the beanCounterQueue")
	return nil
}

// queueForSelf puts the song information on the queue to re-trigger this lambda
func queueForSelf(s *types.Song, nextUpdated *time.Time) error {
	// now
	l, _ := time.LoadLocation(location)
	now := time.Now().In(l).UTC().Unix()

	// queue delay
	diff := nextUpdated.Unix() - now
	var delaySeconds int64
	if diff <= 0 {
		// don't set it to 0 or the lambda will trigger over and over rapidly
		delaySeconds = 10
	} else {
		delaySeconds = diff
	}

	// create message body
	mb, _ := json.Marshal(types.ChuneRefreshBody{SongID: s.SongID})
	input := &sqs.SendMessageInput{
		DelaySeconds: &delaySeconds,
		MessageBody:  aws.String(string(mb)),
		QueueUrl:     aws.String(chuneRefreshQueue),
	}

	// send the message to the queue
	message, err := clients.SQSClient.SendMessage(input)

	if err != nil {
		logger.Log.Error().Err(err).Str("songID", s.SongID).Msg("Unable to put the song onto the refreshQueue")
		return err
	}

	logger.Log.Info().Str("songID", s.SongID).Str("messageID", *message.MessageId).Msg("Successfully put the song onto the refreshQueue")
	logger.Log.Info().Msg(fmt.Sprintf("See you in %d seconds", delaySeconds))
	return nil
}

// lookupSpotify queries Spotify for a song to obtain info like its ID and album art etc
func lookupSpotify(s *types.Song) error {
	// authenticate to spotify
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_SECRET_ID"),
		TokenURL:     spotify.TokenURL,
	}
	token, err := config.Token(context.Background())
	if err != nil {
		logger.Log.Error().Err(err).Msg("Couldn't get spotify token")
		return err
	}
	client := spotify.Authenticator{}.NewClient(token)

	// search spotify for the track
	logger.Log.Info().Str("track", s.SearchString()).Msg("Searching spotify for track")
	results, searchErr := client.SearchOpt(s.SearchString(), spotify.SearchTypeTrack, &spotify.Options{
		Limit: aws.Int(3),
	})
	if searchErr != nil {
		logger.Log.Error().Err(err).Str("track", s.SearchString()).Msg("Unable to search spotify for track")
		return searchErr
	}

	// search returned no results
	if len(results.Tracks.Tracks) == 0 {
		msg := "spotify search returned no results"
		logger.Log.Warn().Str("track", s.SearchString()).Msg(msg)
		return errors.New(msg)
	}

	// search returned results
	songID := results.Tracks.Tracks[0].SimpleTrack.ID.String()
	logger.Log.Info().Str("songID", songID).Msg("Found song on spotify!")
	s.SongID = songID

	return nil
}

// getNowPlaying queries JJJ to see what is getting played right now
func getNowPlaying() (*types.Song, *time.Time) {
	logger.Log.Info().Msg("Checking JJJ for what's playing")

	// query JJJ
	nowPlaying, getNowErr := http.Get(
		fmt.Sprintf("https://music.abcradio.net.au/api/v1/plays/triplej/now.json?tz=%s", location),
	)
	if getNowErr != nil {
		logger.Log.Error().Err(getNowErr).Msg("Couldn't get latest song")
		return nil, nil
	}

	defer nowPlaying.Body.Close()

	// unmarshal response
	response := jjj.ResponseBody{}
	bodyBytes, _ := ioutil.ReadAll(nowPlaying.Body)
	jsonErr := json.Unmarshal(bodyBytes, &response)
	if jsonErr != nil {
		logger.Log.Error().Err(jsonErr).Msg("Unable to unmarshal JJJ response to jjjResponseBody")
		return nil, nil
	}

	// parse the nextUpdated time
	nextUpdated, _ := time.Parse(time.RFC3339, response.NextUpdated)

	// the arid is an empty string when nothing is playing
	if response.Now.Arid == "" {
		logger.Log.Info().Msg("Nothing is currently playing")
		return nil, &nextUpdated
	}

	title := response.Now.Recording.Title
	logger.Log.Info().Str("song", title).Msg("There is a song currently playing")

	// get when the song was played
	var playedAt string
	playedTime, timeParseErr := time.Parse(time.RFC3339, response.Now.PlayedTime)
	if timeParseErr != nil {
		logger.Log.Warn().Msg("Unable to parse JJJ PlayedTime to RFC3339, using now instead")
		playedAt = time.Now().UTC().Format(time.RFC3339)
	} else {
		playedAt = playedTime.Format(time.RFC3339)
	}

	// get artwork
	var artwork []jjj.ArtworkSize
	if len(response.Now.Release.Artwork) == 0 || response.Now.Release == nil {
		artwork = nil
	} else {
		artwork = response.Now.Release.Artwork[0].Sizes
	}

	return &types.Song{
		Name:     title,
		Album:    response.Now.Release.Title,
		Artist:   response.Now.Release.Artists[0].Name,
		Artwork:  &artwork,
		PlayedAt: &playedAt,
	}, &nextUpdated
}

func HandleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	// unmarshall sqsEvent to messageBody
	mb := types.ChuneRefreshBody{}
	jsonErr := json.Unmarshal([]byte(sqsEvent.Records[0].Body), &mb)
	if jsonErr != nil {
		logger.Log.Error().Err(jsonErr).Msg("Unable to unmarshal sqsEvent body to messageBody struct")
		return jsonErr
	}

	// get what's now playing on JJJ
	jjjSong, nextUpdated := getNowPlaying()

	// if no song is playing just put the song back on the queue with the same ID
	if jjjSong == nil {
		logger.Log.Info().Str("songID", mb.SongID).Msg("Putting song back on queue with updated delay as no new song is playing yet")
		_ = queueForSelf(&types.Song{SongID: mb.SongID}, nextUpdated)
		return nil
	}

	// lookup song on Spotify
	spotifyLookupErr := lookupSpotify(jjjSong)
	if spotifyLookupErr != nil {
		logger.Log.Warn().Str("songID", mb.SongID).Msg("Putting song back on queue because we couldn't search for it on spotify")
		_ = queueForSelf(&types.Song{SongID: mb.SongID}, nextUpdated)
		return nil
	}

	// if the same song is playing then come back later
	if jjjSong.SongID == mb.SongID {
		logger.Log.Info().Str("songID", mb.SongID).Msg("Putting song back on queue as it is still playing")
		_ = queueForSelf(jjjSong, nextUpdated)
		return nil
	}

	// add the song to the table if it doesn't exist
	exists, _ := jjjSong.Exists()
	if !exists {
		_ = jjjSong.Create()
	}

	// mark the song as played
	_ = jjjSong.Played()

	// trigger scorer lambda
	_ = queueForCounter(&jjjSong.SongID)

	// queueForSelf self trigger
	_ = queueForSelf(jjjSong, nextUpdated)

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
