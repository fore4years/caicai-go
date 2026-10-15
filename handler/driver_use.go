package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ DriverUseController（/driverUse） ============

type DriverUseController struct{}

// Save GET /driverUse/save 对齐 Java DriverUseServiceImpl.save：预约次数固定 3，日期为当天。
func (c *DriverUseController) Save(ctx *gin.Context) {
	openid := ctx.Query("openid")
	rec := model.DriveruseTbl{
		Openid:        openid,
		ReserveNumber: 3,
		EveryDay:      timePtr(time.Now()),
	}
	repository.Create(&rec)
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}
