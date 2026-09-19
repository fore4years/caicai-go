package main

import (
	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/middleware"
	"caicai-go/router"
	"strconv"

	"github.com/gin-gonic/gin"
)

var serverPort = ":" + strconv.Itoa(int(conf.Cfg.App.Port))

func main() {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.MiddleLog(logger.Mylog))

	router.RouterInit(r)

	r.Run(serverPort)
}
