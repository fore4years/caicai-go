package service

import (
	"caicai-go/protocol"
)

// CmdBody 指令 JSON 载荷。字段命名与取值由业务方约定（TODO：占位）。
type CmdBody struct {
	Ctl  int `json:"ctl"`
	Cmd  int `json:"cmd"`
	Data int `json:"data"`
}

var (
	noneSocket int = 0 // 用于开枪锁指令,使用处没有插座概念,设为0
)

// OpenGauge 开闸（开启继电器）。data 为插座号（如 "1"），对齐 Java MessageFactory.openGauge(imei, "0", data)。
func OpenGauge(imei string, data int) bool {
	return SendCommand(imei, int(protocol.OpOpenGauge), data)
}

// CloseGauge 关闸（关闭继电器）。data 为插座号，对齐 Java MessageFactory.closeGauge(imei, "0", data)。
func CloseGauge(imei string, data int) bool {
	return SendCommand(imei, int(protocol.OpCloseGauge), data)
}

// ClosePlaceLock 关闭车位锁
func ClosePlaceLock(imei string) bool {
	return SendCommand(imei, int(protocol.OpClosePlaceLock), noneSocket)
}

// OpenPlaceLock 开车位锁
func OpenPlaceLock(imei string) bool {
	return SendCommand(imei, int(protocol.OpOpenPlaceLock), noneSocket)
}

// OpenGunLock 开枪锁
func OpenGunLock(imei string) bool {
	return SendCommand(imei, int(protocol.OpOpenCharge), noneSocket)
}

// directionData 将方向字符串映射为下行指令的 data 值：L=-1、R=-2、D=0（MQTT JSON 用）。
func directionData(dir string) int {
	switch dir {
	case "R":
		return -2
	case "L":
		return -1
	default:
		return 0
	}
}

// OpenGunLockForYX 开驿享充电枪锁。dir 为 "L"/"R"（私桩 "D" 不处理）。
func OpenGunLockForYX(imei, dir string) bool {
	if dir == "D" {
		return false
	}
	return SendCommand(imei, int(protocol.OpOpenCharge), directionData(dir))
}

// OpenEle 电表合闸（驿享充电）。dir 为 "L"/"R"。
func OpenEle(imei, dir string) bool {
	return SendCommand(imei, int(protocol.OpOpenEle), directionData(dir))
}

// CloseEle 电表拉闸（驿享充电）。dir 为 "L"/"R"。
func CloseEle(imei, dir string) bool {
	return SendCommand(imei, int(protocol.OpCloseEle), directionData(dir))
}

// ============ 查询类指令（无 data） ============

// OpenLock 开充电桩箱锁。
func OpenLock(imei string) bool {
	return SendCommand(imei, int(protocol.OpOpenLock), -1)
}

// GetInfo 获取设备详细信息。
func GetInfo(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetInfo), -1)
}

// GetPower 读取用电量。
func GetPower(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetPower), -1)
}

// GetElePower 获取电表功率。
func GetElePower(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetElePower), -1)
}

// GetEleCurrent 获取电网电流。
func GetEleCurrent(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetEleCurrent), -1)
}

// GetEleVoltage 获取电网电压。
func GetEleVoltage(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetEleVoltage), -1)
}

// GetEleFrequency 获取电网频率。
func GetEleFrequency(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetEleFrequency), -1)
}

// GetEleData 获取电网数据。
func GetEleData(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetEleData), -1)
}

// GetChargeGunStatus 获取充电枪状态。
func GetChargeGunStatus(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetChargeGunStatus), -1)
}

// GetGunHolderStatus 获取枪座状态。
func GetGunHolderStatus(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetGunHolderStatus), -1)
}

// GetVersion 获取软件版本号。
func GetVersion(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetVersion), -1)
}

// GetTemperature 读取温度。
func GetTemperature(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetTemperature), -1)
}

// GetPlaceLock 获取车位锁。
func GetPlaceLock(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetPlaceLock), -1)
}

// GetICCID 获取 ICCID。
func GetICCID(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetICCID), -1)
}

// GetGaugePower 读取继电器电量。
func GetGaugePower(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetGaugePower), -1)
}

// GetGaugePowerAll 获取所有通道电量。
func GetGaugePowerAll(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetGaugePowerAll), -1)
}

// GetBMS 车载 BMS Ready。
func GetBMS(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetBMS), -1)
}

// GetPrivateData 获取私桩数据。
func GetPrivateData(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetPrivateData), -1)
}

// GetDoubleGunData 获取双枪计量数据。
func GetDoubleGunData(imei string) bool {
	return SendCommand(imei, int(protocol.OpGetDoubleGunData), -1)
}

// ============ 控制类指令（单字节 data 或无 data） ============

// SetPwm 设置 PWM 占空比。
func SetPwm(imei string, pwm int) bool {
	return SendCommand(imei, int(protocol.OpSetPWM), pwm)
}

// OpenVoice 播放提示语音。
func OpenVoice(imei string, data int) bool {
	return SendCommand(imei, int(protocol.OpOpenVoice), data)
}

// RotationPlatform 旋转平台控制。
func RotationPlatform(imei string, data int) bool {
	return SendCommand(imei, int(protocol.OpRotationPlatform), data)
}

// ChargingEndDetection 结束订单断电检测。
func ChargingEndDetection(imei string) bool {
	return SendCommand(imei, int(protocol.OpChargingEndDetection), -1)
}

// CloseOpenTime 关闭车位牌显示。
func CloseOpenTime(imei string) bool {
	return SendCommand(imei, int(protocol.OpCloseOpenTime), -1)
}

// StartOpenTime 开启车位牌显示。
func StartOpenTime(imei string) bool {
	return SendCommand(imei, int(protocol.OpStartOpenTime), -1)
}

// OpenGunHolderSensor 打开枪座压感开关。
func OpenGunHolderSensor(imei string) bool {
	return SendCommand(imei, int(protocol.OpOpenGunHolderSensor), -1)
}

// PowerLimitOn 功率限制（开）。
func PowerLimitOn(imei string) bool {
	return SendCommand(imei, int(protocol.OpPowerLimitOn), -1)
}

// PowerLimitOff 功率限制（关）。
func PowerLimitOff(imei string) bool {
	return SendCommand(imei, int(protocol.OpPowerLimitOff), -1)
}

// OpenFireAlarm 播放火警警报。
func OpenFireAlarm(imei string) bool {
	return SendCommand(imei, int(protocol.OpOpenFireAlarm), -1)
}

// ============ 多字节 data 指令 ============

// SetOpenTime 设置空闲时段。
func SetOpenTime(imei string, data []byte) bool {
	return SendCommandBytes(imei, int(protocol.OpOpenTime), data)
}

// ChangeOpenTime 修改车位牌显示时间。
func ChangeOpenTime(imei string, data []byte) bool {
	return SendCommandBytes(imei, int(protocol.OpChangeOpenTime), data)
}
