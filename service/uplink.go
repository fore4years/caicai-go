package service

import (
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"caicai-go/logger"
	"caicai-go/protocol"
	"caicai-go/repository"
)

// directionString 将方向字节映射为 Java getDirectionString 的 "D"/"L"/"R"。
func directionString(ctl byte) string {
	switch ctl {
	case protocol.DirectionLeft:
		return "L"
	case protocol.DirectionRight:
		return "R"
	default:
		return "D"
	}
}

// handleDeviceUplink 处理上行设备数据：按 cmd 缓存到对应用途的 Redis 键，对齐 Java MessageListeningServiceImpl。
func handleDeviceUplink(deviceID string, ctl, cmd byte, data []byte) {
	nid := strings.ToUpper(deviceID)
	dir := directionString(ctl)
	dataStr := string(data)

	switch cmd {
	case protocol.OpGetElePower: // 电表功率 → key = nid+方向
		if v, err := decimal.NewFromString(dataStr); err == nil {
			SetRedisValue(nid+dir, v.Div(decimal.NewFromInt(10)).Round(2).String())
		}
	case protocol.OpGetEleCurrent: // 电流 → key = nid+方向:current
		if v, err := decimal.NewFromString(dataStr); err == nil {
			SetRedisValue(nid+dir+":current", v.Div(decimal.NewFromInt(1000)).Round(2).String())
		}
	case protocol.OpGetEleVoltage: // 电压 → key = nid+方向:voltage
		if v, err := decimal.NewFromString(dataStr); err == nil {
			SetRedisValue(nid+dir+":voltage", v.Div(decimal.NewFromInt(10)).Round(2).String())
		}
	case protocol.OpGetPrivateData: // 私桩数据 → key = nid+方向
		SetRedisValue(nid+dir, dataStr)
	case protocol.OpGetGunHolderStatus: // 枪座状态 → key = nid:gun_holder_status
		SetRedisValue(nid+":gun_holder_status", dataStr)
	case protocol.OpGetChargeGunStatus: // 充电枪状态 → key = nid+方向:charge_gun_status
		SetRedisValue(nid+dir+":charge_gun_status", dataStr)
	case protocol.OpGetGaugePower: // 二轮车继电器电量 → 更新插座电量 + 订单电量
		onGetGaugePower(nid, dataStr)
	case protocol.OpGetGaugePowerAll: // 所有继电器电量 → 更新各通道插座电量
		onGetAllGaugePower(nid, dataStr)
	default:
		handleOrderedUplink(nid, cmd, dataStr)
	}
}

// handleOrderedUplink 处理需要按订单查询的上行事件（距离检测、充满、插枪/还枪、BMS）。
func handleOrderedUplink(nid string, cmd byte, dataStr string) {
	order := repository.FindUseOrderByDevice(nid)
	if order == nil {
		return
	}

	switch cmd {
	case protocol.OpChargingPileDistance: // 充电桩检测距离
		SetRedisValue("chargingPileDetection:orderId:"+order.Orderid, dataStr)
	case protocol.OpPlaceLockDistance: // 车位锁检测距离
		SetRedisValue("placeLockDistance:orderId:"+order.Orderid, dataStr)
	case protocol.OpGetBMS: // 车载 BMS Ready → 充电中
		SetRedisValue("orderState:"+order.Orderid, "充电中")
	case protocol.OpPowerFull: // 电量充满
		if order.FullTime == nil || order.FullTime.IsZero() {
			repository.UpdateOrderFullTime(order.Orderid)
		}
		if order.ConsumptionType == "邻享充电" {
			SetRedisValue("PrivateOrderState:"+order.Orderid, "充电完成")
		} else {
			SetRedisValue("orderState:"+order.Orderid, "充电完成")
		}
	case protocol.OpChargeGunPull: // 充电枪已拔出
		repository.UpdateOrderChargeGunPull(order.Orderid, 1)
		SetRedisValue("orderState:"+order.Orderid, "插枪入车")
	case protocol.OpChargeGunBack: // 充电枪已归还
		repository.UpdateOrderChargeGunPull(order.Orderid, 0)
		SetRedisValue("orderState:"+order.Orderid, "充电完成")
	}
}

// onGetGaugePower 读取二轮车继电器电量（对齐 Java onGetGaugePower）。
// data 形如 "通道,电量,电流,电压,功率,频率,电流有效值,电压有效值,功率有效值,温度"。
func onGetGaugePower(nid, dataStr string) {
	parts := strings.Split(dataStr, ",")
	if len(parts) < 5 {
		return
	}
	channel := parts[0]
	meterValue, err := decimal.NewFromString(parts[1])
	if err != nil {
		return
	}
	power := parts[4]

	product := repository.ProductByDeviceID(nid)
	if product == nil {
		return
	}
	repository.UpsertReceptaclePower(product.Pid, channel, meterValue)

	order := repository.FindTwiceOrderByPidAndChannel(product.Pid, channel)
	if order == nil {
		return
	}
	repository.UpdateReOrderPower(order.Orderid, meterValue, power)
}

// onGetAllGaugePower 读取所有继电器电量（对齐 Java onGetAllGaugePower）。
// data 形如 "电量1,电量2,..."，每个 i 对应插座通道 i+1。
func onGetAllGaugePower(nid, dataStr string) {
	product := repository.ProductByDeviceID(nid)
	if product == nil {
		return
	}
	for i, s := range strings.Split(dataStr, ",") {
		if v, err := decimal.NewFromString(strings.TrimSpace(s)); err == nil {
			repository.UpsertReceptaclePower(product.Pid, strconv.Itoa(i+1), v)
		}
	}
}

// mqttDeviceReport MQTT 上报的 JSON 载荷（topic = device/{imei}/report）。
type mqttDeviceReport struct {
	Event        string          `json:"event"`
	Channel      int             `json:"channel"`
	Relay        int             `json:"relay"`
	VoltageV     float64         `json:"voltage_v"`
	CurrentA     float64         `json:"current_a"`
	PowerW       decimal.Decimal `json:"power_w"`
	EnergyKwh    decimal.Decimal `json:"energy_kwh"`
	FrequencyHz  float64         `json:"frequency_hz"`
	TemperatureC float64         `json:"temperature_c"`
}

// extractDeviceFromTopic 从 topic（device/{imei}/report）中提取设备标识（imei/nid）。
func extractDeviceFromTopic(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// handleMqttReport 处理 MQTT 上报：更新插座电量与订单电量（对齐 Java onGetGaugePower 的数据来源）。
func handleMqttReport(deviceID string, report mqttDeviceReport) {
	product := repository.ProductByDeviceID(deviceID)
	if product == nil {
		logger.Mylog.Debug().Msgf("MQTT 上报未找到产品, device: %s", deviceID)
		return
	}
	channel := strconv.Itoa(report.Channel)
	repository.UpsertReceptaclePower(product.Pid, channel, report.EnergyKwh)

	order := repository.FindTwiceOrderByPidAndChannel(product.Pid, channel)
	if order == nil {
		return
	}
	repository.SetReOrderPower(order.Orderid, report.EnergyKwh, report.PowerW)
}
