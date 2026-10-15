package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/repository"
)

// ============ SubscribeMessageController（/subscribeMessage） ============

type SubscribeMessageController struct{}

// SelectRid 查询 rid（微信接口，省略外呼，返回空串）。
func (c *SubscribeMessageController) SelectRid(ctx *gin.Context) {
	ctx.String(http.StatusOK, "")
}

func (c *SubscribeMessageController) SendParkingMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendRelocateCarMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendParkingTicketMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendChargingMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendChargingSettleMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendChargingExceptionMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendOrderOverMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}

// SendMoveCarMessage 对齐 Java sendMoveCarMessage：更新 move_car_time。
func (c *SubscribeMessageController) SendMoveCarMessage(ctx *gin.Context) {
	var req struct {
		CarNumber string `json:"carNumber"`
		Phone     string `json:"phone"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	user, err := repository.UserByPhone(ctx.Request.Context(), req.Phone)
	if err != nil || user.Openid == "" {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if latest, err := repository.LatestNeighborOrderByOpenid(user.Openid); err == nil {
		repository.UpdateOrderMoveCarTime(latest.Orderid)
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}
