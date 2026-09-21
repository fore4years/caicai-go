// Package protocol 实现下行指令的协议编码，对齐 Java 的 Message.toBytes() 与 OpCodeConstants。
package protocol

import "encoding/hex"

// 帧头 / 版本 / 结束符（对齐 Java Message 接口常量）
const (
	Header    = 0xAA
	Version   = 0x01 // LoRa
	Version4G = 0x02
	End       = 0xCD
)

// 方向（对齐 Java Message.Direction）
const (
	DirectionDefault = 0x00
	DirectionLeft    = 0x11
	DirectionRight   = 0x22
)

// 命令字（对齐 Java OpCodeConstants）
const (
	OpGatewayRegister  = 0xFF
	OpSetGateway       = 0x00
	OpSetFactory       = 0x01
	OpOpenLock         = 0xA2 // 开充电桩箱锁
	OpOpenCharge       = 0xA3 // 开启充电
	OpCloseEle         = 0xA4 // 电表拉闸
	OpOpenPlaceLock    = 0xA6 // 开启车位锁
	OpClosePlaceLock   = 0xA7 // 关闭车位锁
	OpGetPower         = 0xA8
	OpOpenEle          = 0xA9 // 电表合闸
	OpGetElePower      = 0xB2
	OpSetPID           = 0xB9
	OpOpenTime         = 0xBA
	OpBluetooth        = 0xBB
	OpBluetoothPass    = 0xBD
	OpOpenGauge        = 0xC0 // 开启继电器
	OpCloseGauge       = 0xC1 // 关闭继电器
	OpGetGaugePower    = 0xC2
	OpGetGaugePowerAll = 0xC4
	OpOpenVoice        = 0x00 // 占位，实际 0xCA
)

// Encode 构造下行指令字节，对齐 Java Message.toBytes()。
//
// 帧格式：
//   - LoRa（nid ≤ 4 字节）：[nid] AA 01 dir opcode len(2,大端) data crc CD
//   - 4G（nid > 7 字节，IMEI）：AA 02 dir opcode len(2,大端) data crc CD（不带 nid）
//
// crc 为 payload（AA 到 data 结束）的字节求和，对齐 Java CRCUtils.getCRC8。
func Encode(nid string, opcode, direction byte, data []byte) []byte {
	nidBytes, _ := hex.DecodeString(nid)

	var ver byte = Version
	if len(nidBytes) > 7 {
		ver = Version4G
	}

	// payload = AA ver dir opcode len(2) data
	payload := []byte{Header, ver, direction, opcode, byte(len(data) >> 8), byte(len(data))}
	payload = append(payload, data...)

	var out []byte
	if len(nidBytes) <= 7 {
		out = append(out, nidBytes...) // LoRa 帧头带 nid
	}
	out = append(out, payload...)
	out = append(out, crc8(payload))
	out = append(out, End)
	return out
}

// crc8 字节求和（对齐 Java CRCUtils.getCRC8）。
func crc8(b []byte) byte {
	var sum int
	for _, v := range b {
		sum += int(v)
	}
	return byte(sum)
}
