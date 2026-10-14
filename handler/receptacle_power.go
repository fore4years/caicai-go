package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ ReceptaclePowerController（/receptacle/power） ============

type ReceptaclePowerController struct{}

func (c *ReceptaclePowerController) GetPowerByPid(ctx *gin.Context) {
	var rows []model.ReceptaclePowerTbl
	conf.Db.Where("pid = ?", ctx.Query("pid")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *ReceptaclePowerController) GetPowerByReceptacleId(ctx *gin.Context) {
	var rows []model.ReceptaclePowerTbl
	conf.Db.Where("receptacle_id = ?", ctx.Query("ReceptacleId")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}
