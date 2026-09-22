package logger

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

var Mylog zerolog.Logger

func init() {
	if gin.IsDebugging() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		// 开发环境：带颜色的 ConsoleWriter，人肉友好
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		Mylog = zerolog.New(output).With().Timestamp().Logger()
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		Mylog = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

}
