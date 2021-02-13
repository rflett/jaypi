package log

import (
	"github.com/rs/zerolog"
	dl "github.com/rs/zerolog/log"
	"os"
)

var (
	environment = "dev"
	Log         = zerolog.Logger{}
)

func init() {
	// sets up the log
	if v, ok := os.LookupEnv("STAGE"); ok {
		environment = v
	}

	// in dev we use the pretty console output, other envs use JSON
	if os.Getenv("STAGE") == "dev" || os.Getenv("STAGE") == "" {
		output := zerolog.ConsoleWriter{Out: os.Stdout}
		Log = zerolog.New(output).Hook(SeverityHook{}).With().Timestamp().Logger()
		dl.Level(zerolog.DebugLevel)
	} else {
		Log = dl.Hook(SeverityHook{}).With().Str("host", os.Getenv("SERVICE")).Str("environment", environment).Logger()
		dl.Level(zerolog.InfoLevel)
	}

	dl.Level(zerolog.InfoLevel)
}

type SeverityHook struct{}

func (h SeverityHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if level != zerolog.NoLevel {
		e.Str("severity", level.String())
	}
}
