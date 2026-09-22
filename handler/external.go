package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
	"caicai-go/protocol"
	"caicai-go/service"
)

// ExternalController 对齐 Java controller.ExternalApiController（外部 API，HMAC 鉴权）。
type ExternalController struct{}

// OpenGunHolderSensor GET /external/openGunHolderSensor?imei=
func (e *ExternalController) OpenGunHolderSensor(c *gin.Context) {
	imei := c.Query("imei")

	// 1. 根据 Imei 查 imei_tab 中 charging_station = imei 的记录，获取 chargingSocket
	var row model.ImeiTab
	if err := conf.Db.Where("charging_station = ?", imei).First(&row).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(404, "未找到该 Imei 对应的枪座绑定关系"))
		return
	}
	chargingSocket := row.ChargingSocket

	// 2. chargingSocket 作为设备标识下发 openGunHolderSensor 指令
	ok := service.SendCommand(chargingSocket, protocol.OpOpenGunHolderSensor, protocol.DirectionDefault, nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}
