package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/repository"
)

// ============ ReceptaclePowerController（/receptacle/power） ============

type ReceptaclePowerController struct{}

func (c *ReceptaclePowerController) GetPowerByPid(ctx *gin.Context) {
	rows := repository.ReceptaclePowersByPid(ctx.Query("pid"))
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *ReceptaclePowerController) GetPowerByReceptacleId(ctx *gin.Context) {
	rows := repository.ReceptaclePowersByReceptacleID(ctx.Query("ReceptacleId"))
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}
