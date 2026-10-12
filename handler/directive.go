package handler

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

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
	case "1", "11", "0x11", "LEFT", "left", "L", "l":
		return protocol.DirectionLeft
	case "2", "22", "0x22", "RIGHT", "right", "R", "r":
		return protocol.DirectionRight
	default:
		return protocol.DirectionDefault
	}
}

// reverseHexString 反转字符串，按 2 个十六进制字符为一组倒序（对齐 Java HexUtil.reverseHexString）。
func reverseHexString(s string) string {
	var b strings.Builder
	for i := len(s) - 2; i >= 0; i -= 2 {
		b.WriteString(s[i : i+2])
	}
	return b.String()
}

// convertStringToHex 将字符串转为 ASCII 十六进制（对齐 Java HexUtil.convertStringToHex）。
func convertStringToHex(s string) string {
	return hex.EncodeToString([]byte(s))
}

// parseTimeData 解析车位牌时间数据 "HH:MM-HH:MM"，转为 4 字节（对齐 Java ChangeOpenTimeMessage.parseTimeData）。
func parseTimeData(timeData string) []byte {
	if timeData == "" {
		return []byte{0, 0, 0, 0}
	}
	parts := strings.SplitN(timeData, "-", 2)
	if len(parts) != 2 {
		return []byte{0, 0, 0, 0}
	}
	start := strings.SplitN(parts[0], ":", 2)
	end := strings.SplitN(parts[1], ":", 2)
	if len(start) != 2 || len(end) != 2 {
		return []byte{0, 0, 0, 0}
	}
	sh, e1 := strconv.Atoi(start[0])
	sm, e2 := strconv.Atoi(start[1])
	eh, e3 := strconv.Atoi(end[0])
	em, e4 := strconv.Atoi(end[1])
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return []byte{0, 0, 0, 0}
	}
	return []byte{byte(sh), byte(sm), byte(eh), byte(em)}
}

// SendHex /directive/sendHex?gateway=&hex=
func (d *DirectiveController) SendHex(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(service.SendHex(c.Query("gateway"), c.Query("hex"))))
}

// OpenLock /directive/openLock?gateway=&nid=
func (d *DirectiveController) OpenLock(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpOpenLock), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenPlaceLock /directive/openPlaceLock
func (d *DirectiveController) OpenPlaceLock(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpOpenPlaceLock), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// ClosePlaceLock /directive/closePlaceLock
func (d *DirectiveController) ClosePlaceLock(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpClosePlaceLock), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenCharge /directive/openCharge?gateway=&nid=&direction=
func (d *DirectiveController) OpenCharge(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpOpenCharge), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetInfo /directive/getInfo?gateway=&nid=&direction=
func (d *DirectiveController) GetInfo(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetInfo), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetPower /directive/getPower
func (d *DirectiveController) GetPower(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetPower), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetElePower /directive/getElePower（SSE 降级为 JSON）
func (d *DirectiveController) GetElePower(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetElePower), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetEleCurrent /directive/getEleCurrent
func (d *DirectiveController) GetEleCurrent(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetEleCurrent), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetEleVoltage /directive/getEleVoltage
func (d *DirectiveController) GetEleVoltage(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetEleVoltage), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetEleFrequency /directive/getEleFrequency
func (d *DirectiveController) GetEleFrequency(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetEleFrequency), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetEleData /directive/getEleData
func (d *DirectiveController) GetEleData(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetEleData), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// SetGatewayAddress /directive/setGatewayAddress?gateway=&nid=&gatewayId=
// 对应 _OpSetGateway（下划线前缀命令，按约定不实现）。
func (d *DirectiveController) SetGatewayAddress(c *gin.Context) {
	gatewayID := c.Query("gatewayId")
	if len(gatewayID) != 8 {
		c.JSON(http.StatusOK, ResultError(400, "gatewayId长度限制为8"))
		return
	}
	if _, err := hex.DecodeString(gatewayID); err != nil {
		c.JSON(http.StatusOK, ResultError(400, "gatewayId长度限制为8"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(false))
}

// SetPwm /directive/setPwm?gateway=&nid=&direction=&pwm=
func (d *DirectiveController) SetPwm(c *gin.Context) {
	pwm, _ := strconv.Atoi(c.Query("pwm"))
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpSetPWM), pwm)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetGatewayList /directive/getGatewayList（MQTT 下暂无在线列表，返回空）
func (d *DirectiveController) GetGatewayList(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess([]string{}))
}

// SetOpenTime /directive/setOpenTime?gateway=&nid=&ledMac=&OpenTime=&opCode=
func (d *DirectiveController) SetOpenTime(c *gin.Context) {
	dataHex := convertStringToHex(reverseHexString(c.Query("ledMac"))) + convertStringToHex(c.Query("OpenTime"))
	data, _ := hex.DecodeString(dataHex)
	ok := service.SendCommandBytes(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpOpenTime), data)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// CalibrationPower /directive/CalibrationPower?gateway=&value1=&value2=
func (d *DirectiveController) CalibrationPower(c *gin.Context) {
	v1, _ := strconv.Atoi(c.Query("value1"))
	v2, _ := strconv.Atoi(c.Query("value2"))
	calibrationValue1 := strings.ToUpper(strconv.FormatInt(int64(v1), 16))
	calibrationValue2 := strings.ToUpper(strconv.FormatInt(int64(v2), 16))
	hexStr := "AA0200" + "C3" + "0004" + calibrationValue1 + calibrationValue2 + "45CD"
	ok := service.SendHex(c.Query("gateway"), hexStr)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenGauge /directive/openGauge?gateway=&nid=&data=
func (d *DirectiveController) OpenGauge(c *gin.Context) {
	v, _ := strconv.Atoi(c.Query("data"))
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpOpenGauge), v)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// CloseGauge /directive/closeGauge?gateway=&nid=&data=
func (d *DirectiveController) CloseGauge(c *gin.Context) {
	v, _ := strconv.Atoi(c.Query("data"))
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpCloseGauge), v)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenEle /directive/OpenEle?gateway=&nid=&direction=
func (d *DirectiveController) OpenEle(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpOpenEle), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// CloseEle /directive/closeEle?gateway=&nid=&direction=
func (d *DirectiveController) CloseEle(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpCloseEle), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenVoice /directive/openVoice?gateway=&nid=&data=
func (d *DirectiveController) OpenVoice(c *gin.Context) {
	v, _ := strconv.Atoi(c.Query("data"))
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpOpenVoice), v)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// RotationPlatform /directive/rotationPlatform?gateway=&nid=&data=
func (d *DirectiveController) RotationPlatform(c *gin.Context) {
	v, _ := strconv.Atoi(c.Query("data"))
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpRotationPlatform), v)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// ChargingEndDetection /directive/chargingEndDetection?gateway=&nid=&direction=
func (d *DirectiveController) ChargingEndDetection(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpChargingEndDetection), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// SendBluetoothMac /directive/sendBluetoothMac?gateway=&nid=&mac=
// 对应 _OpSendBluetoothMac（下划线前缀命令，按约定不实现）。
func (d *DirectiveController) SendBluetoothMac(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(false))
}

// BluetoothMacCancel /directive/BluetoothMacCancel?gateway=&nid=
// 对应 _OpBluetoothMacCancel（下划线前缀命令，按约定不实现）。
func (d *DirectiveController) BluetoothMacCancel(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(false))
}

// SetRedisValue /directive/setRedisValue?key=&value=
func (d *DirectiveController) SetRedisValue(c *gin.Context) {
	service.SetRedisValue(c.Query("key"), c.Query("value"))
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// GetChargeGunStatus /directive/getChargeGunStatus?gateway=&nid=&direction=
func (d *DirectiveController) GetChargeGunStatus(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetChargeGunStatus), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// ChangeOpenTime /directive/changeOpenTime?gateway=&nid=&timeData=
func (d *DirectiveController) ChangeOpenTime(c *gin.Context) {
	ok := service.SendCommandBytes(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpChangeOpenTime), parseTimeData(c.Query("timeData")))
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// CloseOpenTime /directive/closeOpenTime?gateway=&nid=
func (d *DirectiveController) CloseOpenTime(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpCloseOpenTime), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// StartOpenTime /directive/startOpenTime?gateway=&nid=
func (d *DirectiveController) StartOpenTime(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpStartOpenTime), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetGunHolderStatus /directive/getGunHolderStatus?gateway=&nid=
func (d *DirectiveController) GetGunHolderStatus(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetGunHolderStatus), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenGunHolderSensor /directive/openGunHolderSensor?gateway=&nid=
func (d *DirectiveController) OpenGunHolderSensor(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpOpenGunHolderSensor), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// ============================ 其余导出（无下划线前缀）的下行查询/控制命令 ============================
// Java DirectiveController 未暴露这些 HTTP 端点（多由其它服务/定时任务直接调用），此处补齐对齐协议命令字。

// GetVersion /directive/getVersion
func (d *DirectiveController) GetVersion(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetVersion), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetTemperature /directive/getTemperature
func (d *DirectiveController) GetTemperature(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetTemperature), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetPlaceLock /directive/getPlaceLock
func (d *DirectiveController) GetPlaceLock(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetPlaceLock), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetICCID /directive/getICCID
func (d *DirectiveController) GetICCID(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetICCID), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetGaugePower /directive/getGaugePower
func (d *DirectiveController) GetGaugePower(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetGaugePower), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetGaugePowerAll /directive/getGaugePowerAll
func (d *DirectiveController) GetGaugePowerAll(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetGaugePowerAll), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetBMS /directive/getBMS
func (d *DirectiveController) GetBMS(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetBMS), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// PowerLimitOn /directive/powerLimitOn
func (d *DirectiveController) PowerLimitOn(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpPowerLimitOn), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// PowerLimitOff /directive/powerLimitOff
func (d *DirectiveController) PowerLimitOff(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpPowerLimitOff), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// OpenFireAlarm /directive/openFireAlarm
func (d *DirectiveController) OpenFireAlarm(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpOpenFireAlarm), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetPrivateData /directive/getPrivateData
func (d *DirectiveController) GetPrivateData(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetPrivateData), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}

// GetDoubleGunData /directive/getDoubleGunData
func (d *DirectiveController) GetDoubleGunData(c *gin.Context) {
	ok := service.SendCommand(deviceID(c.Query("gateway"), c.Query("nid")), int(protocol.OpGetDoubleGunData), -1)
	c.JSON(http.StatusOK, ResultSuccess(ok))
}
