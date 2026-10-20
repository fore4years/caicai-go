package service

import "caicai-go/protocol"

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

// OpenGunLock 开枪锁
func OpenGunLock(imei string) bool {
	return SendCommand(imei, int(protocol.OpOpenCharge), noneSocket)
}
