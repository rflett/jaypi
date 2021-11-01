package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
	"io/ioutil"
	"jjj.rflett.com/jjj-api/clients"
	"jjj.rflett.com/jjj-api/logger"
	"jjj.rflett.com/jjj-api/services"
	"jjj.rflett.com/jjj-api/types"
	"jjj.rflett.com/jjj-api/types/jjj"
	"net/http"
	"os"
	"time"
)

const tzLocation = "Australia/Sydney"

var (
	chuneRefreshQueue = os.Getenv("REFRESH_QUEUE")
	beanCounterQueue  = os.Getenv("COUNTER_QUEUE")
	config            = &clientcredentials.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_SECRET_ID"),
		TokenURL:     spotify.TokenURL,
	}
	client = spotify.Client{}
)

// queueForCounter puts the songID on a queue to trigger the counter lambdas
func queueForCounter(songID *string) error {
	// create message body
	mb, _ := json.Marshal(types.BeanCounterBody{SongID: *songID})
	input := &sqs.SendMessageInput{
		DelaySeconds: 0,
		MessageBody:  aws.String(string(mb)),
		QueueUrl:     &beanCounterQueue,
	}

	// send the message to the queue
	message, err := clients.SQSClient.SendMessage(context.TODO(), input)

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
	location, _ := time.LoadLocation(tzLocation)
	now := time.Now().In(location).UTC().Unix()

	// queue delay
	diff := nextUpdated.Unix() - now
	var delaySeconds int32
	if diff <= 0 {
		// don't set it to 0 or the lambda will trigger over and over rapidly
		delaySeconds = 10
	} else {
		delaySeconds = int32(diff)
	}

	// create message body
	mb, _ := json.Marshal(types.ChuneRefreshBody{SongID: s.SongID})
	input := &sqs.SendMessageInput{
		DelaySeconds: delaySeconds,
		MessageBody:  aws.String(string(mb)),
		QueueUrl:     &chuneRefreshQueue,
	}

	// send the message to the queue
	message, err := clients.SQSClient.SendMessage(context.TODO(), input)

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
	// search spotify for the track
	logger.Log.Info().Str("track", s.SearchString()).Msg("Searching spotify for track")
	results, err := client.SearchOpt(s.SearchString(), spotify.SearchTypeTrack, &spotify.Options{
		Limit: aws.Int(3),
	})
	if err != nil {
		logger.Log.Error().Err(err).Str("track", s.SearchString()).Msg("Unable to search spotify for track")
		return err
	}

	// search returned no results
	if results.Tracks.Total == 0 {
		msg := "spotify search returned no results"
		logger.Log.Warn().Str("track", s.SearchString()).Msg(msg)
		return errors.New(msg)
	}

	// search returned results, update song with spotify info
	s.SongID = results.Tracks.Tracks[0].SimpleTrack.ID.String()
	s.Name = results.Tracks.Tracks[0].SimpleTrack.Name
	s.Album = results.Tracks.Tracks[0].Album.Name
	s.Artist = results.Tracks.Tracks[0].Artists[0].Name

	// get the album artwork
	var artwork []jjj.ArtworkSize
	for _, art := range results.Tracks.Tracks[0].Album.Images {
		artwork = append(artwork, jjj.ArtworkSize{
			Url:    art.URL,
			Width:  art.Width,
			Height: art.Height,
		})
	}
	s.Artwork = &artwork
	logger.Log.Info().Str("songID", s.SongID).Msg("Found song on spotify!")

	return nil
}

// getNowPlaying queries JJJ to see what is getting played right now
func getNowPlaying() (*types.Song, *time.Time) {
	logger.Log.Info().Msg("Checking JJJ for what's playing")

	// query JJJ
	nowPlaying, err := http.Get(
		fmt.Sprintf("https://music.abcradio.net.au/api/v1/plays/triplej/now.json?tz=%s", tzLocation),
	)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Couldn't get latest song")
		return nil, nil
	}

	defer nowPlaying.Body.Close()

	// unmarshal response
	response := jjj.ResponseBody{}
	bodyBytes, _ := ioutil.ReadAll(nowPlaying.Body)
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to unmarshal JJJ response to jjjResponseBody")
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
		logger.Log.Warn().Msg("Unable to parse JJJ PlayedTime to RFC3339, using current time instead")
		playedAt = time.Now().UTC().Format(time.RFC3339)
	} else {
		playedAt = playedTime.Format(time.RFC3339)
	}

	return &types.Song{
		Name:     title,
		Album:    response.Now.Release.Title,
		Artist:   response.Now.Release.Artists[0].Name,
		PlayedAt: &playedAt,
	}, &nextUpdated
}

func HandleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	// unmarshall sqsEvent to messageBody
	body := types.ChuneRefreshBody{}
	err := json.Unmarshal([]byte(sqsEvent.Records[0].Body), &body)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Unable to unmarshal sqsEvent body to messageBody struct")
		return err
	}

	// get what's now playing on JJJ
	jjjSong, nextUpdated := getNowPlaying()

	// if no song is playing just put the song back on the queue with the same ID
	if jjjSong == nil {
		logger.Log.Info().Str("songID", body.SongID).Msg("Putting song back on queue with updated delay as no new song is playing yet")
		return queueForSelf(&types.Song{SongID: body.SongID}, nextUpdated)
	}

	// lookup song on Spotify
	if err = lookupSpotify(jjjSong); err != nil {
		logger.Log.Warn().Str("songID", body.SongID).Msg("Putting song back on queue because we couldn't search for it on spotify")
		return queueForSelf(&types.Song{SongID: body.SongID}, nextUpdated)
	}

	// if the same song is playing then come back later
	if jjjSong.SongID == body.SongID {
		logger.Log.Info().Str("songID", body.SongID).Msg("Putting song back on queue as it is still playing")
		return queueForSelf(&types.Song{SongID: body.SongID}, nextUpdated)
	}

	// add the song to the table if it doesn't exist
	exists, _ := jjjSong.Exists()
	if !exists {
		_ = jjjSong.Create()
	}

	// get the play count
	currentPlayCount, _ := services.GetCurrentPlayCount()

	// mark the song as played
	_ = jjjSong.Played(currentPlayCount)

	// trigger scorer lambda
	_ = queueForCounter(&jjjSong.SongID)

	// queueForSelf self trigger
	_ = queueForSelf(jjjSong, nextUpdated)

	return nil
}

func init() {
	token, _ := config.Token(context.Background())
	client = spotify.Authenticator{}.NewClient(token)
}

func main() {
	lambda.Start(HandleRequest)
}
