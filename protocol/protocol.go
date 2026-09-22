// Package protocol 实现设备指令的协议编解码，对齐 Java 的 Message.toBytes() 与 OpCodeConstants。
// 帧格式（MQTT 调用资料_增强版.docx）：
//   - 下行（云端→设备）：[0xAA] [0x02版本] [ctl] [cmd] [len_h] [len_l] [data...] [校验和] [0xCD]
//   - 上行（设备→云端）：[IMEI 8字节] [0xAA] [0x02版本] [ctl] [cmd] [len_h] [len_l] [data...] [校验和] [0xCD]
package protocol

import (
	"encoding/hex"
	"errors"
)

// 帧头 / 版本 / 结束符。
const (
	Header    = 0xAA
	Version   = 0x01 // LoRa
	Version4G = 0x02
	End       = 0xCD
)

// 方向 / 控制位（对齐 Java Message.Direction）。
const (
	DirectionDefault = 0x00
	DirectionLeft    = 0x11
	DirectionRight   = 0x22
)

// 命令字（对齐 Java OpCodeConstants）。
const (
	OpSetGateway           = 0x00 // 设置网关 mac
	OpSetFactory           = 0x01 // 设置出厂蓝牙 mac
	OpGetVersion           = 0xA0 // 获取软件版本号
	OpSetPWM               = 0xA1 // 设置 PWM 占空比
	OpOpenLock             = 0xA2 // 开充电桩箱锁
	OpOpenCharge           = 0xA3 // 开启充电
	OpCloseEle             = 0xA4 // 电表拉闸
	OpGetTemperature       = 0xA5 // 读取温度
	OpOpenPlaceLock        = 0xA6 // 开启车位锁
	OpClosePlaceLock       = 0xA7 // 关闭车位锁
	OpGetPower             = 0xA8 // 读取用电量
	OpOpenEle              = 0xA9 // 电表合闸
	OpGetPlaceLock         = 0xAA // 获取车位锁
	OpButtonDown           = 0xAB // 紧急按钮按下
	OpButtonUp             = 0xAC // 紧急按钮复位
	OpChargeGunBack        = 0xAD // 充电枪已归还
	OpChargeGunPull        = 0xAE // 充电枪已拔出
	OpGetInfo              = 0xAF // 获取设备详细信息
	OpPowerFull            = 0xB0 // 电量已充满
	OpChargeGunDisconnect  = 0xB1 // 充电枪与车辆断开
	OpGetElePower          = 0xB2 // 获取电表功率
	OpGetEleVoltage        = 0xB3 // 获取电网电压
	OpGetEleCurrent        = 0xB4 // 获取电网电流
	OpGetEleFrequency      = 0xB5 // 获取电网频率
	OpChargeGunConnect     = 0xB6 // 充电枪与车辆连接
	OpGetEleData           = 0xB7 // 获取电网数据
	OpGetICCID             = 0xB8 // 获取 ICCID
	OpSetPID               = 0xB9 // 设置 PID
	OpOpenTime             = 0xBA // 空闲时段
	OpBluetooth            = 0xBB // 蓝牙请求开车位锁
	OpBluetoothCharge      = 0xBC // 蓝牙请求充电
	OpBluetoothPass        = 0xBD // 蓝牙请求通过
	OpBluetoothFail        = 0xBE // 蓝牙请求失败
	OpBluetoothSendFail    = 0xBF // 蓝牙发送失败
	OpOpenGauge            = 0xC0 // 开启继电器
	OpCloseGauge           = 0xC1 // 关闭继电器
	OpGetGaugePower        = 0xC2 // 读取继电器电量
	OpCalibrationPower     = 0xC3 // 校准电量
	OpGetGaugePowerAll     = 0xC4 // 获取所有通道电量
	OpGetBMS               = 0xC5 // 车载 BMS Ready
	OpPowerLimitOn         = 0xC6 // 功率限制（开）
	OpPowerLimitOff        = 0xC7 // 功率限制（关）
	OpOpenFireAlarm        = 0xC8 // 播放火警警报
	OpPlaceLockDistance    = 0xC9 // 车位锁检测距离
	OpOpenVoice            = 0xCA // 播放提示语音
	OpGetPrivateData       = 0xCB // 获取私桩数据
	OpGetDoubleGunData     = 0xCC // 获取双枪计量数据
	OpRotationPlatform     = 0xCE // 旋转平台控制
	OpChargingPileDistance = 0xCF // 充电桩检测距离
	OpChangeOpenTime       = 0xD0 // 修改车位牌显示时间
	OpGetChargeGunStatus   = 0xD1 // 获取充电枪状态
	OpCloseOpenTime        = 0xD3 // 关闭车位牌显示
	OpStartOpenTime        = 0xD4 // 开启车位牌显示
	OpChargingEndDetection = 0xDC // 结束订单断电检测
	OpSendBluetoothMac     = 0xDD // 传输蓝牙 mac 地址
	OpBluetoothMacSuccess  = 0xDE // 蓝牙 mac 匹配成功
	OpBluetoothMacCancel   = 0xDF // 蓝牙 mac 取消匹配
	OpGetGunHolderStatus   = 0xF3 // 获取枪座状态与电量
	OpOpenGunHolderSensor  = 0xF6 // 打开枪座压感开关
	OpGatewayRegister      = 0xFF // 网关注册
)

// EncodeCmd 构造下行指令字节（云端→设备），帧格式：
// [0xAA] [0x02] [ctl] [cmd] [len_h] [len_l] [data...] [crc] [0xCD]。
// ctl 对应 Java 的 direction，cmd 对应 opcode。
func EncodeCmd(cmd, ctl byte, data []byte) []byte {
	payload := []byte{Header, Version4G, ctl, cmd, byte(len(data) >> 8), byte(len(data))}
	payload = append(payload, data...)
	out := append(payload, crc8(payload))
	out = append(out, End)
	return out
}

// Encode 构造下行指令字节（兼容旧调用）。
// 4G（nid > 7 字节）用 Version4G；LoRa（nid ≤ 4 字节）用 Version 并在帧头带 nid。
// crc 为 payload（AA 到 data 结束）的字节求和，对齐 Java CRCUtils.getCRC8。
func Encode(nid string, opcode, direction byte, data []byte) []byte {
	nidBytes, _ := hex.DecodeString(nid)

	var ver byte = Version
	if len(nidBytes) > 7 {
		ver = Version4G
	}

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

// Decode 解析上行帧（设备→云端）：
// [IMEI/NID 前缀][0xAA][版本][ctl][cmd][len_h][len_l][data...][crc][0xCD]。
// 返回设备标识（IMEI/NID 的 hex 字符串）、ctl、cmd 与 data。
func Decode(b []byte) (deviceID string, ctl, cmd byte, data []byte, err error) {
	if len(b) < 8 {
		return "", 0, 0, nil, errors.New("帧过短")
	}
	// 定位帧头 0xAA，之前为设备标识前缀。
	idx := -1
	for i, v := range b {
		if v == Header {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", 0, 0, nil, errors.New("未找到帧头 0xAA")
	}
	if idx > 0 {
		deviceID = hex.EncodeToString(b[:idx])
	}

	frame := b[idx:]
	if len(frame) < 9 { // AA ver ctl cmd len_h len_l crc CD 最少 8 字节 + data
		return deviceID, 0, 0, nil, errors.New("帧体过短")
	}
	if frame[len(frame)-1] != End {
		return deviceID, 0, 0, nil, errors.New("帧尾非 0xCD")
	}

	ver := frame[1]
	_ = ver
	ctl = frame[2]
	cmd = frame[3]
	length := int(frame[4])<<8 | int(frame[5])

	bodyEnd := 6 + length
	if bodyEnd+2 > len(frame) {
		return deviceID, ctl, cmd, nil, errors.New("长度字段越界")
	}
	data = frame[6:bodyEnd]
	if frame[bodyEnd] != crc8(frame[:bodyEnd]) {
		return deviceID, ctl, cmd, data, errors.New("校验和不匹配")
	}
	return deviceID, ctl, cmd, data, nil
}

// crc8 字节求和（对齐 Java CRCUtils.getCRC8）。
func crc8(b []byte) byte {
	var sum int
	for _, v := range b {
		sum += int(v)
	}
	return byte(sum)
}
