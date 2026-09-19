package router

import (
	"caicai-go/handler"

	"github.com/gin-gonic/gin"
)


func RouterInit(r *gin.Engine) {
	api := r.Group("/user")
	{
		userController := new(handler.UserController)
		api.POST("/login", userController.AddUser)
	}
}