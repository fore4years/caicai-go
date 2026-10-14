package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
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
	var user model.UserTbl
	if err := conf.Db.Where("phone = ?", req.Phone).First(&user).Error; err != nil || user.Openid == "" {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	var latest model.OrderTbl
	if err := conf.Db.Where("openid = ? AND consumption_type = ? AND state = ?", user.Openid, "邻享充电", "使用中").
		Order("begin_time desc").First(&latest).Error; err == nil {
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", latest.Orderid).Update("move_car_time", time.Now())
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}
