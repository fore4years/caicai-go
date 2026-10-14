package router

import (
	"caicai-go/handler"
	"net/http"

	"github.com/gin-gonic/gin"
)

func healthyPing(c *gin.Engine) {
	c.Any("ping", healthyPingHandler)
}

func healthyPingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, handler.ResultSuccess(true))
}
