package sentry

import (
	sentryGo "github.com/getsentry/sentry-go"
	"jjj.rflett.com/jjj-api/logger"
	"os"
	"time"
)

func init() {
	err := sentryGo.Init(sentryGo.ClientOptions{
		Debug:      false,
		ServerName: os.Getenv("FUNCTION_NAME"),
	})
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Could not init sentry")
	}
	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentryGo.Flush(2 * time.Second)
}
