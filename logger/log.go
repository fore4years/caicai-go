package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var Mylog zerolog.Logger

func init() {
	Mylog = zerolog.New(os.Stdout).With().Timestamp().Logger()
}