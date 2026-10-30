package service

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"caicai-go/logger"
	"caicai-go/model"
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

// bcdToInt 将 BCD 字节解码为整数（每字节两 BCD 数字），对齐 Java ByteUtils.bcdToInt。
func bcdToInt(b []byte) int {
	result := 0
	for _, v := range b {
		result = result*100 + int((v&0xf0)>>4)*10 + int(v&0x0f)
	}
	return result
}

// meterFromGunData 解析拔枪/还枪上行电量（对齐 Java ChangeGunPull/BackMessage.getMeterValue）。
// 4G（nid>12）返回逗号分隔的第 2 段 * 100；否则按 BCD 解码。
func meterFromGunData(nid string, data []byte) int32 {
	if parts := strings.Split(string(data), ","); len(parts) > 2 && len(nid) > 12 {
		if v, err := decimal.NewFromString(parts[1]); err == nil {
			return int32(v.Mul(decimal.NewFromInt(100)).Round(0).IntPart())
		}
	}
	if len(data) == 0 || len(data) > 4 {
		return -1
	}
	return int32(bcdToInt(data))
}

// handleDeviceUplink 处理上行设备数据：按 cmd 缓存/落库，对齐 Java MessageListeningServiceImpl。
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
	case protocol.OpGetPrivateData: // 私桩数据 → 更新电量 + Redis 参数
		onGetPrivateData(nid, dir, dataStr)
	case protocol.OpGetGunHolderStatus: // 枪座状态 → key = nid:gun_holder_status
		SetRedisValue(nid+":gun_holder_status", dataStr)
	case protocol.OpGetChargeGunStatus: // 充电枪状态 → key = nid+方向:charge_gun_status
		SetRedisValue(nid+dir+":charge_gun_status", dataStr)
	case protocol.OpGetGaugePower: // 二轮车继电器电量 → 更新插座电量 + 订单电量
		onGetGaugePower(nid, dataStr)
	case protocol.OpGetGaugePowerAll: // 所有继电器电量 → 更新各通道插座电量
		onGetAllGaugePower(nid, dataStr)
	case protocol.OpButtonDown: // 紧急按钮按下
		onButtonDown(nid, dir)
	case protocol.OpButtonUp: // 紧急按钮复位
		onButtonUp(nid, dir)
	case protocol.OpChargeGunDisconnect: // 充电枪与车辆断开
		onChargeGunDisconnect(nid)
	case protocol.OpGatewayRegister: // 网关注册（TCP 0xFF）
		onGatewayRegister(nid, data)
	default:
		handleOrderedUplink(nid, dir, cmd, data, dataStr)
	}
}

// handleOrderedUplink 处理需要按订单查询的上行事件（距离检测、充满、插枪/还枪、BMS、电表示数）。
func handleOrderedUplink(nid, dir string, cmd byte, data []byte, dataStr string) {
	order := repository.FindUseOrderByDevice(nid)
	if order == nil {
		order = repository.FindPrivateUseOrderByDevice(nid)
	}
	if order == nil {
		order = repository.FindPrivateReturnGunOrderByDevice(nid)
	}
	if order == nil {
		logger.Mylog.Debug().Msgf("未找到设备对应订单, device=%s cmd=0x%02X", nid, cmd)
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
	case protocol.OpGetPower: // 电表示数
		onGetPower(nid, dir, data, order)
	case protocol.OpGetDoubleGunData: // 双枪计量数据
		onGetDoubleGunData(nid, dir, dataStr, order)
	case protocol.OpChargeGunPull: // 充电枪已拔出
		onChargeGunPull(nid, dir, data, order)
	case protocol.OpChargeGunBack: // 充电枪已归还
		onChargeGunBack(nid, dir, data, order)
	}
}

// onGetPower 返回电表示数：更新充电枪表底 + 订单电子电量（对齐 Java onGetPower）。
func onGetPower(nid, dir string, data []byte, order *model.OrderTbl) {
	meter := int32(bcdToInt(data))
	repository.UpdateChargingGunMeter(nid, dir, meter)
	repository.SaveOrUpdateOrderElectronicMeter(order.Orderid, meter)
}

// onGetDoubleGunData 双枪计量数据：更新订单电子电量 + Redis 功率/电流/电压（对齐 Java onGetDoubleGunData）。
func onGetDoubleGunData(nid, dir, dataStr string, order *model.OrderTbl) {
	parts := strings.Split(dataStr, ",")
	if len(parts) < 5 {
		return
	}
	meterValue, err := decimal.NewFromString(parts[1])
	if err != nil {
		return
	}
	meter := meterValue.Mul(decimal.NewFromInt(100)).Round(0)
	repository.SaveOrUpdateOrderElectronicMeter(order.Orderid, int32(meter.IntPart()))

	SetRedisValue(nid+dir, parts[4])            // 功率
	SetRedisValue(nid+dir+":current", parts[2]) // 电流
	SetRedisValue(nid+dir+":voltage", parts[3]) // 电压
}

// onGetPrivateData 私桩数据：更新订单电子电量 + Redis 功率/电压/电流（对齐 Java onGetPrivateData）。
func onGetPrivateData(nid, dir, dataStr string) {
	parts := strings.Split(dataStr, ",")
	if len(parts) < 5 {
		return
	}
	meterValue, err := decimal.NewFromString(parts[1])
	if err != nil {
		return
	}
	meter := meterValue.Mul(decimal.NewFromInt(100)).Round(0)

	if order := repository.FindPrivateUseOrderByDevice(nid); order != nil {
		repository.SaveOrUpdateOrderElectronicMeter(order.Orderid, int32(meter.IntPart()))
	}

	// 小程序所需参数：功率(W, 除以1000保留1位)、电压、电流（精度对齐 Java onGetPrivateData）
	power, _ := decimal.NewFromString(parts[4])
	powerValue := power.Div(decimal.NewFromInt(1000)).Round(1).InexactFloat64()
	voltage, _ := decimal.NewFromString(parts[3])
	voltageValue := voltage.Round(2).InexactFloat64()
	current, _ := decimal.NewFromString(parts[2])
	currentValue := current.Round(2).InexactFloat64()
	m := map[string]float64{
		"power":   powerValue,
		"voltage": voltageValue,
		"current": currentValue,
	}
	b, _ := json.Marshal(m)
	SetRedisValue(nid+dir, string(b))
}

// onChargeGunPull 充电枪拔出：更新表底 + 新增时段电量 + Redis（对齐 Java onChangeGunPull）。
func onChargeGunPull(nid, dir string, data []byte, order *model.OrderTbl) {
	meter := meterFromGunData(nid, data)
	logger.Mylog.Info().Msgf("充电枪拔出: device=%s dir=%s order=%s meter=%d", nid, dir, order.Orderid, meter)
	repository.UpdateChargingGunMeter(nid, dir, meter)
	repository.UpdateOrderChargeGunPull(order.Orderid, 1)
	repository.AddOrderElectronic(order.Orderid, meter)
	SetRedisValue("orderState:"+order.Orderid, "插枪入车")

	// 驿享充电且无车位锁时，以拔枪时间作为开始时间
	if order.ConsumptionType == "驿享充电" {
		if spaces := repository.SpacesByID(order.SpacesID); spaces != nil && spaces.LockID == "" {
			repository.UpdateOrderBeginTime(order.Orderid)
		}
	}
}

// onChargeGunBack 充电枪归还：更新表底 + 结束时间/充电时长/充电度数/电子电量 + Redis。
// 对齐 Java onChangeGunBack → handleGunReturnOrderCompletion 的数据落库部分（订单扣费结束仍走 HTTP）。
func onChargeGunBack(nid, dir string, data []byte, order *model.OrderTbl) {
	meter := meterFromGunData(nid, data)
	logger.Mylog.Info().Msgf("充电枪归还: device=%s dir=%s order=%s meter=%d", nid, dir, order.Orderid, meter)
	repository.UpdateChargingGunMeter(nid, dir, meter)
	repository.UpdateOrderChargeGunPull(order.Orderid, 0)
	repository.UpdateOrderOverTime(order.Orderid)
	repository.UpdateOrderChargingTime(order.Orderid)
	repository.UpdateChargingDegreeForOrder(order.Orderid)
	repository.SaveOrUpdateOrderElectronicMeter(order.Orderid, meter)
	SetRedisValue("orderState:"+order.Orderid, "充电完成")

	// 还枪后自动结束订单（对齐 Java handleGunReturnOrderCompletion）。
	if FinishGunReturnOrderFn != nil {
		FinishGunReturnOrderFn(order.Orderid, meter)
	}
}

// onButtonDown 紧急按钮按下：写入 charging_state_tbl（对齐 Java onButtonDown）。
func onButtonDown(nid, dir string) {
	product := repository.ProductByDeviceID(nid)
	if product == nil {
		return
	}
	gun := repository.ChargingGunByPidAndDirection(product.Pid, dir)
	if gun == nil {
		return
	}
	address, phone, ownership := "", "", ""
	if gun.Gun.Direction == "D" { // 私桩
		if place, _ := repository.PrivatePlaceByChargingGunID(gun.Gun.ID); place != nil {
			address = place.Place
			if bean := repository.PrivateChargingBySpacesNum(place.SpacesCode); bean != nil {
				ownership = bean.Name
				phone = bean.Phone
			}
		}
	} else {
		if spaces := repository.SpacesByChargingGunID(gun.Gun.ID); spaces != nil {
			address = spaces.Place
		}
	}
	if repository.ChargingStatePressedExists(product.Pid, dir) {
		return
	}
	repository.InsertChargingState(&model.ChargingStateTbl{
		ProductID:            product.Pid,
		Direction:            dir,
		EmergencyButtonState: "Pressed",
		Model:                strconv.Itoa(int(gun.Gun.Model)),
		ProductOwnership:     ownership,
		Address:              address,
		Phone:                phone,
		Area:                 extractArea(address),
	})
	logger.Mylog.Info().Msgf("紧急按钮按下: device=%s dir=%s pid=%s address=%s", nid, dir, product.Pid, address)
}

// onButtonUp 紧急按钮复位：删除 charging_state_tbl（对齐 Java onButtonUp）。
func onButtonUp(nid, dir string) {
	product := repository.ProductByDeviceID(nid)
	if product == nil {
		return
	}
	repository.DeleteChargingState(product.Pid, dir)
	logger.Mylog.Info().Msgf("紧急按钮复位: device=%s dir=%s pid=%s", nid, dir, product.Pid)
}

// onChargeGunDisconnect 充电枪与车辆断开：记录断开时间（对齐 Java onChangeGunDisconnect）。
func onChargeGunDisconnect(nid string) {
	if order := repository.FindUseOrderByDevice(nid); order != nil {
		repository.UpdateOrderGunDisconnectTime(order.Orderid)
		logger.Mylog.Info().Msgf("枪车断开: device=%s order=%s", nid, order.Orderid)
	}
}

// onGatewayRegister 网关注册（TCP 0xFF）：按 4G/LoRa 区分 mac/iccid，不存在则入库。
// 对齐 Java GatewayServiceImpl.onMessage。
func onGatewayRegister(nid string, data []byte) {
	var mac, iccid string
	if len(nid) > 8 {
		// 4G：mac = 设备标识（imei），data 形如 "iccid81.69.46.160..."，取分隔符前段作为 iccid
		mac = nid
		iccid = strings.Split(string(data), "81.69.46.160")[0]
	} else {
		// LoRa：mac = data 的 hex
		mac = strings.ToUpper(hex.EncodeToString(data))
	}
	if repository.GatewayByMac(mac) != nil {
		logger.Mylog.Debug().Msgf("网关已注册, 跳过: mac=%s", mac)
		return
	}
	if err := repository.InsertGateway(mac, iccid); err != nil {
		logger.Mylog.Err(err).Msgf("网关注册入库失败: mac=%s, iccid=%s", mac, iccid)
		return
	}
	logger.Mylog.Info().Msgf("网关注册成功: mac=%s, iccid=%s", mac, iccid)
}

// extractArea 从地址中提取「X区」（对齐 Java 正则 (?<=[省市区])[^省市区]+(?=区)）。
func extractArea(address string) string {
	re := regexp.MustCompile(`[省市区]([^省市区]+)区`)
	if m := re.FindStringSubmatch(address); m != nil {
		return m[1] + "区"
	}
	return ""
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
		logger.Mylog.Debug().Msgf("二轮车电量上报未找到产品, device=%s", nid)
		return
	}

	// 电压 > 250V 判定设备故障，主动关闸（对齐 Java onGetGaugePower）
	if v, err := strconv.ParseFloat(parts[3], 64); err == nil && v > 250.0 {
		logger.Mylog.Warn().Msgf("设备故障: 电压 %s > 250V, 自动关闸, pid=%s, 通道=%s", parts[3], product.Pid, channel)
		if rid, e := strconv.Atoi(channel); e == nil {
			CloseGauge(product.Imei, rid)
		}
	}

	repository.UpsertReceptaclePower(product.Pid, channel, meterValue)

	order := repository.FindTwiceOrderByPidAndChannel(product.Pid, channel)
	if order == nil {
		logger.Mylog.Debug().Msgf("二轮车电量上报未找到使用中订单, pid=%s channel=%s", product.Pid, channel)
		return
	}
	// 功率异常值（>10000）跳过（硬件偶发异常，对齐 Java）
	if p, e := strconv.ParseFloat(power, 64); e == nil && p > 10000 {
		return
	}
	repository.UpdateReOrderPower(order.Orderid, meterValue, power)

	// 充满自停模式（未选择定时）：功率连续低时判定充满并自动结束（对齐 Java onGetGaugePower → GaugeFull）
	if order.TwiceChargeTime == "" {
		GaugeFull(order.Orderid)
	}
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
	Channel     int32   `json:"Channel"`
	Current     float64 `json:"Current"`
	Voltage     float64 `json:"Voltage"`
	Power       float64 `json:"Power"`
	Temperature float64 `json:"Temperature"`
	Energy      float64 `json:"Energy"`
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
	logger.Mylog.Info().Msgf(
		"device: %s ====> 通道: %d 电流: %.2f 电压: %.2f 功率: %.2f 电量: %.2f",
		deviceID, report.Channel, report.Current, report.Voltage, report.Power, report.Energy,
	)
	switch report.Channel {
	case 0:
		// 邻享充电
		if privateOrder := repository.FindPrivateUseOrderByDevice(deviceID); privateOrder != nil {
			// 对齐 Java handleNeighbor：key = nid+D，值 {power,voltage,current}（功率 W→kW 保留1位，电压/电流保留2位）
			nid := strings.ToUpper(deviceID)
			chargeData := map[string]float64{
				"power":   decimal.NewFromFloat(report.Power).Div(decimal.NewFromInt(1000)).Round(1).InexactFloat64(),
				"voltage": decimal.NewFromFloat(report.Voltage).Round(2).InexactFloat64(),
				"current": decimal.NewFromFloat(report.Current).Round(2).InexactFloat64(),
			}
			b, _ := json.Marshal(chargeData)
			SetRedisValue(nid+"D", string(b))
			logger.Mylog.Info().Msgf("[MQTT] 邻享充电写入Redis key=%s order=%s", nid+"D", privateOrder.Orderid)

			meter := decimal.NewFromFloat(report.Energy).Mul(decimal.NewFromInt(100)).Round(0)
			repository.SaveOrUpdateOrderElectronicMeter(privateOrder.Orderid, int32(meter.IntPart()))
		} else {
			logger.Mylog.Warn().Msgf("[MQTT] 邻享充电未找到使用中订单, 未写入Redis, device=%s", deviceID)
		}
	case -1, -2:
		// 驿享充电：-1=左枪(L)、-2=右枪(R)。写功率/电流/电压到 nid+direction（与读取端 getMeterValue/getCurrentValue/getVoltageValue 一致）
		dir := "D"
		if report.Channel == -1 {
			dir = "L"
		} else if report.Channel == -2 {
			dir = "R"
		}
		nid := strings.ToUpper(deviceID)
		SetRedisValue(nid+dir, decimal.NewFromFloat(report.Power).String())
		SetRedisValue(nid+dir+":current", decimal.NewFromFloat(report.Current).String())
		SetRedisValue(nid+dir+":voltage", decimal.NewFromFloat(report.Voltage).String())
		logger.Mylog.Info().Msgf("[MQTT] 驿享充电写入Redis nid=%s dir=%s power=%.2f current=%.2f voltage=%.2f", nid, dir, report.Power, report.Current, report.Voltage)

		// 落库电表值到 tab_order_electronic（对齐 Java onGetDoubleGunData：nid+方向查使用中订单，电量 setScale(2)×100）
		if order := repository.FindYixiangUseOrderByNidDirection(nid, dir); order != nil {
			meter := decimal.NewFromFloat(report.Energy).Round(2).Mul(decimal.NewFromInt(100))
			repository.SaveOrUpdateOrderElectronicMeter(order.Orderid, int32(meter.IntPart()))
			logger.Mylog.Info().Msgf("[MQTT] 驿享充电写入电表值 order=%s meter=%d", order.Orderid, int32(meter.IntPart()))
		} else {
			logger.Mylog.Warn().Msgf("[MQTT] 驿享充电未找到使用中订单, 未写入电表值, device=%s dir=%s", deviceID, dir)
		}

	default:
		// 二轮车充电
		channel := fmt.Sprintf("%d", report.Channel)
		power := decimal.NewFromFloat(report.Power)
		meterValue := decimal.NewFromFloat(report.Energy)
		repository.UpsertReceptaclePower(product.Pid, channel, meterValue)

		// 通过 imei::channel 映射订单号（开闸时 SetMap 写入），内存映射丢失时回退 DB 反查并回填
		orderID, ok := GetMap(deviceID + "::" + channel)
		if !ok {
			order := repository.FindTwiceOrderByPidAndChannel(product.Pid, channel)
			if order == nil {
				logger.Mylog.Warn().Msgf("[MQTT] 二轮车未找到使用中订单, 未写入Redis, device=%s channel=%s", deviceID, channel)
				return
			}
			orderID = order.Orderid
			SetMap(deviceID+"::"+channel, orderID)
		}

		// 计费电量落库
		repository.SetReOrderPower(orderID, meterValue, power)

		// 前端展示：按订单号缓存最新上报数据
		reportJSON, _ := json.Marshal(report)
		SetRedisValue("orderReport:"+orderID, string(reportJSON))
		logger.Mylog.Info().Msgf("[MQTT] 二轮车写入Redis orderReport:%s channel=%s", orderID, channel)
	}
}
