package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/protocol"
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
	order := findUseOrderByDevice(nid)
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
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", order.Orderid).Update("full_time", time.Now())
		}
		if order.ConsumptionType == "邻享充电" {
			SetRedisValue("PrivateOrderState:"+order.Orderid, "充电完成")
		} else {
			SetRedisValue("orderState:"+order.Orderid, "充电完成")
		}
	case protocol.OpChargeGunPull: // 充电枪已拔出
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", order.Orderid).Update("charge_gun_pull", 1)
		SetRedisValue("orderState:"+order.Orderid, "插枪入车")
	case protocol.OpChargeGunBack: // 充电枪已归还
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", order.Orderid).Update("charge_gun_pull", 0)
		SetRedisValue("orderState:"+order.Orderid, "充电完成")
	}
}

// findUseOrderByDevice 按设备标识（nid，对齐 Java getUseByNid）查询「使用中」订单。
// 设备可能是充电枪（驿享/邻享）或车位锁（踩踩停车），故分别按枪产品与锁产品的 nid 匹配。
func findUseOrderByDevice(deviceID string) *model.OrderTbl {
	var order model.OrderTbl
	err := conf.Db.Raw(`
		SELECT o.* FROM order_tbl o
		LEFT JOIN tab_parking_spaces s ON o.spaces_id = s.id
		LEFT JOIN tab_charging_gun g ON s.charging_gun_id = g.id
		LEFT JOIN tab_product gp ON gp.pid = g.product_id
		LEFT JOIN lock_tbl l ON s.lock_id = l.lockid
		LEFT JOIN tab_product lp ON lp.pid = l.product_id
		WHERE o.state = '使用中'
		  AND (gp.nid = ? OR lp.nid = ?)
		ORDER BY o.reserve_time DESC
		LIMIT 1`, deviceID, deviceID).Scan(&order).Error
	if err != nil || order.Orderid == "" {
		logger.Mylog.Debug().Msgf("未找到设备对应的使用中订单, device: %s", deviceID)
		return nil
	}
	return &order
}

// productByDeviceID 按设备标识查询产品：长度>8 视为 4G IMEI，否则视为 LoRa nid（对齐 Java getByNid）。
func productByDeviceID(deviceID string) *model.TabProduct {
	var p model.TabProduct
	var err error
	if len(deviceID) > 8 {
		err = conf.Db.Where("Imei = ?", deviceID).First(&p).Error
	} else {
		err = conf.Db.Where("nid = ?", deviceID).First(&p).Error
	}
	if err != nil {
		return nil
	}
	return &p
}

// upsertReceptaclePower 更新插座电量（对齐 Java ReceptaclePowerService.addOrUpdate）。
func upsertReceptaclePower(pid, channel string, meterValue decimal.Decimal) {
	cid, _ := strconv.Atoi(channel)
	var existing model.ReceptaclePowerTbl
	if err := conf.Db.Where("pid = ? AND receptacle_id = ?", pid, channel).First(&existing).Error; err != nil {
		conf.Db.Create(&model.ReceptaclePowerTbl{Pid: pid, ReceptacleID: int32(cid), MeterValue: meterValue})
		return
	}
	conf.Db.Model(&model.ReceptaclePowerTbl{}).Where("pid = ? AND receptacle_id = ?", pid, channel).Update("meter_value", meterValue)
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

	product := productByDeviceID(nid)
	if product == nil {
		return
	}
	upsertReceptaclePower(product.Pid, channel, meterValue)

	order := findTwiceOrderByPidAndChannel(product.Pid, channel)
	if order == nil {
		return
	}
	updateReOrderPower(order.Orderid, meterValue, power)
}

// onGetAllGaugePower 读取所有继电器电量（对齐 Java onGetAllGaugePower）。
// data 形如 "电量1,电量2,..."，每个 i 对应插座通道 i+1。
func onGetAllGaugePower(nid, dataStr string) {
	product := productByDeviceID(nid)
	if product == nil {
		return
	}
	for i, s := range strings.Split(dataStr, ",") {
		if v, err := decimal.NewFromString(strings.TrimSpace(s)); err == nil {
			upsertReceptaclePower(product.Pid, strconv.Itoa(i+1), v)
		}
	}
}

// findTwiceOrderByPidAndChannel 查询使用中的二轮车订单（对齐 Java onGetGaugePower 的订单查询）。
func findTwiceOrderByPidAndChannel(pid, channel string) *model.OrderTbl {
	var order model.OrderTbl
	if err := conf.Db.Where("pid = ? AND lockid = ? AND state = ?", pid, channel, "使用中").First(&order).Error; err != nil {
		return nil
	}
	return &order
}

// updateReOrderPower 更新订单充电电量（覆盖式，对齐 Java onGetGaugePower 的 update：二进制协议的电量是累计表底）。
func updateReOrderPower(orderID string, meterValue decimal.Decimal, power string) {
	newPower, _ := decimal.NewFromString(power)
	conf.Db.Exec(
		"UPDATE re_order_electronic_tbl SET over_value = ?, power = ?, max_power = GREATEST(COALESCE(max_power,0), ?), price_time_id = (SELECT id FROM tab_price_time WHERE start_time < CURTIME() AND over_time >= CURTIME() LIMIT 1) WHERE order_id = ?",
		meterValue, power, newPower, orderID,
	)
}

// setReOrderPower 直接更新订单充电电量（energy_kwh 为当前电量值，直接覆盖 over_value，不累加）。
func setReOrderPower(orderID string, energyKwh, powerW decimal.Decimal) {
	conf.Db.Exec(
		"UPDATE re_order_electronic_tbl SET over_value = ?, power = ?, max_power = GREATEST(COALESCE(max_power,0), ?), price_time_id = (SELECT id FROM tab_price_time WHERE start_time < CURTIME() AND over_time >= CURTIME() LIMIT 1) WHERE order_id = ?",
		energyKwh, powerW.StringFixed(2), powerW, orderID,
	)
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
	product := productByDeviceID(deviceID)
	if product == nil {
		logger.Mylog.Debug().Msgf("MQTT 上报未找到产品, device: %s", deviceID)
		return
	}
	channel := strconv.Itoa(report.Channel)
	upsertReceptaclePower(product.Pid, channel, report.EnergyKwh)

	order := findTwiceOrderByPidAndChannel(product.Pid, channel)
	if order == nil {
		return
	}
	setReOrderPower(order.Orderid, report.EnergyKwh, report.PowerW)
}
