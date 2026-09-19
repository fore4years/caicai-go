package main

import (
	"caicai-go/conf"
	"caicai-go/middleware"
	"caicai-go/router"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

var serverPort = ":" + strconv.Itoa(int(conf.Cfg.App.Port))

func main() {
	logger := zlog.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.MiddleLog(logger))

	router.RouterInit(r)

	r.Run(serverPort)
}
