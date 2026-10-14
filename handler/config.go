package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
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
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&cfg).Error == nil))
}
