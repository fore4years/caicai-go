package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ ConfigController（/config） ============

type ConfigController struct{}

// Save GET /config/save 对齐 Java ConfigController.save：insert 一条 config_tbl。
func (c *ConfigController) Save(ctx *gin.Context) {
	var cfg model.ConfigTbl
	if err := ctx.ShouldBind(&cfg); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&cfg).Error == nil))
}
