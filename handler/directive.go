package handler

import (
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/protocol"
	"caicai-go/service"
)

// DirectiveController 对齐 Java controller.DirectiveController（设备控制指令）。
type DirectiveController struct{}

// deviceID 优先用 nid，回退 gateway（MQTT 下按设备标识发主题）。
func deviceID(gateway, nid string) string {
	if nid != "" {
		return nid
	}
	return gateway
}

// parseDirection 解析方向字符串。
func parseDirection(s string) byte {
	switch s {
	case "1", "11", "0x11", "LEFT", "left":
		return protocol.DirectionLeft
	case "2", "22", "0x22", "RIGHT", "right":
		return protocol.DirectionRight
	default:
		return protocol.DirectionDefault
	}
}

// reverseHexString 反转字符串（对齐 Java HexUtil.reverseHexString）。
func reverseHexString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// convertStringToHex 将字符串转为 ASCII 十六进制（对齐 Java HexUtil.convertStringToHex）。
func convertStringToHex(s string) string {
	return hex.EncodeToString([]byte(s))
}

// SendHex /directive/sendHex?gateway=&hex=
func (d *DirectiveController) SendHex(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(service.SendHex(c.Query("gateway"), c.Query("hex"))))
}

// OpenLock /directive/openLock?gateway=&nid=
func (d *DirectiveController) OpenLock(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpOpenLock, protocol.DirectionDefault, nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenPlaceLock /directive/openPlaceLock
func (d *DirectiveController) OpenPlaceLock(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpOpenPlaceLock, protocol.DirectionDefault, nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// ClosePlaceLock /directive/closePlaceLock
func (d *DirectiveController) ClosePlaceLock(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpClosePlaceLock, protocol.DirectionDefault, nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenCharge /directive/openCharge?gateway=&nid=&direction=
func (d *DirectiveController) OpenCharge(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpOpenCharge, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetInfo /directive/getInfo?gateway=&nid=&direction=
func (d *DirectiveController) GetInfo(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpGetInfo, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetPower /directive/getPower
func (d *DirectiveController) GetPower(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpGetPower, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetElePower /directive/getElePower（SSE 降级为 JSON）
func (d *DirectiveController) GetElePower(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpGetElePower, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetEleCurrent /directive/getEleCurrent
func (d *DirectiveController) GetEleCurrent(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpGetEleCurrent, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetEleVoltage /directive/getEleVoltage
func (d *DirectiveController) GetEleVoltage(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpGetEleVoltage, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetEleFrequency /directive/getEleFrequency
func (d *DirectiveController) GetEleFrequency(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpGetEleFrequency, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetEleData /directive/getEleData
func (d *DirectiveController) GetEleData(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpGetEleData, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// SetGatewayAddress /directive/setGatewayAddress?gateway=&nid=&gatewayId=
func (d *DirectiveController) SetGatewayAddress(c *gin.Context) {
	gatewayID := c.Query("gatewayId")
	if len(gatewayID) != 8 {
		c.JSON(http.StatusOK, ResultError(400, "gatewayId长度限制为8"))
		return
	}
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpSetGateway, protocol.DirectionDefault, []byte(gatewayID))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// SetPwm /directive/setPwm?gateway=&nid=&direction=&pwm=
func (d *DirectiveController) SetPwm(c *gin.Context) {
	pwm := c.Query("pwm")
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpSetPWM, parseDirection(c.Query("direction")), []byte(pwm))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetGatewayList /directive/getGatewayList（MQTT 下暂无在线列表，返回空）
func (d *DirectiveController) GetGatewayList(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess([]string{}))
}

// SetOpenTime /directive/setOpenTime?gateway=&nid=&ledMac=&OpenTime=&opCode=
func (d *DirectiveController) SetOpenTime(c *gin.Context) {
	data := convertStringToHex(reverseHexString(c.Query("ledMac"))) + convertStringToHex(c.Query("OpenTime"))
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpOpenTime, protocol.DirectionDefault, []byte(data))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// CalibrationPower /directive/CalibrationPower?gateway=&value1=&value2=
func (d *DirectiveController) CalibrationPower(c *gin.Context) {
	v1, _ := strconv.Atoi(c.Query("value1"))
	v2, _ := strconv.Atoi(c.Query("value2"))
	data := []byte{byte(v1 >> 8), byte(v1), byte(v2 >> 8), byte(v2)}
	ok := service.SendCommand(c.Query("gateway"), protocol.OpCalibrationPower, protocol.DirectionDefault, data)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenGauge /directive/openGauge?gateway=&nid=&data=
func (d *DirectiveController) OpenGauge(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpOpenGauge, protocol.DirectionDefault, []byte(c.Query("data")))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// CloseGauge /directive/closeGauge?gateway=&nid=&data=
func (d *DirectiveController) CloseGauge(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpCloseGauge, protocol.DirectionDefault, []byte(c.Query("data")))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenEle /directive/OpenEle?gateway=&nid=&direction=
func (d *DirectiveController) OpenEle(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpOpenEle, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// CloseEle /directive/closeEle?gateway=&nid=&direction=
func (d *DirectiveController) CloseEle(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpCloseEle, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenVoice /directive/openVoice?gateway=&nid=&data=
func (d *DirectiveController) OpenVoice(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpOpenVoice, protocol.DirectionDefault, []byte(c.Query("data")))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// RotationPlatform /directive/rotationPlatform?gateway=&nid=&data=
func (d *DirectiveController) RotationPlatform(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpRotationPlatform, protocol.DirectionDefault, []byte(c.Query("data")))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// ChargingEndDetection /directive/chargingEndDetection?gateway=&nid=&direction=
func (d *DirectiveController) ChargingEndDetection(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpChargingEndDetection, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// SendBluetoothMac /directive/sendBluetoothMac?gateway=&nid=&mac=
func (d *DirectiveController) SendBluetoothMac(c *gin.Context) {
	macHex := convertStringToHex(reverseHexString(c.Query("mac")))
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpSendBluetoothMac, protocol.DirectionDefault, []byte(macHex))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// BluetoothMacCancel /directive/BluetoothMacCancel?gateway=&nid=
func (d *DirectiveController) BluetoothMacCancel(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpBluetoothMacCancel, protocol.DirectionDefault, nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// SetRedisValue /directive/setRedisValue?key=&value=
func (d *DirectiveController) SetRedisValue(c *gin.Context) {
	service.SetRedisValue(c.Query("key"), c.Query("value"))
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// GetChargeGunStatus /directive/getChargeGunStatus?gateway=&nid=&direction=
func (d *DirectiveController) GetChargeGunStatus(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpGetChargeGunStatus, parseDirection(c.Query("direction")), nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// ChangeOpenTime /directive/changeOpenTime?gateway=&nid=&timeData=
func (d *DirectiveController) ChangeOpenTime(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpChangeOpenTime, protocol.DirectionDefault, []byte(c.Query("timeData")))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// CloseOpenTime /directive/closeOpenTime?gateway=&nid=
func (d *DirectiveController) CloseOpenTime(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpCloseOpenTime, protocol.DirectionDefault, nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// StartOpenTime /directive/startOpenTime?gateway=&nid=
func (d *DirectiveController) StartOpenTime(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpStartOpenTime, protocol.DirectionDefault, nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetGunHolderStatus /directive/getGunHolderStatus?gateway=&nid=
func (d *DirectiveController) GetGunHolderStatus(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpGetGunHolderStatus, protocol.DirectionDefault, nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenGunHolderSensor /directive/openGunHolderSensor?gateway=&nid=
func (d *DirectiveController) OpenGunHolderSensor(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), protocol.OpOpenGunHolderSensor, protocol.DirectionDefault, nil)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}
