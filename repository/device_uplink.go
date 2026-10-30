package repository

import (
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// 设备上行指令的数据落库（对齐 Java MessageListeningServiceImpl 中 updateMeter /
// saveOrUpdateMeterValue / 紧急按钮 / 枪车断开 等 DB 副作用）。
// ============================================================================

// UpdateChargingGunMeter 按 nid+direction 更新充电枪电表表底。
// nid 长度 > 8 视为 4G IMEI，否则视为 LoRa nid（对齐 Java ChargingGunServiceImpl.updateMeter）。
func UpdateChargingGunMeter(nid, direction string, meter int32) {
	if meter < 0 {
		return
	}
	col := "nid"
	if len(nid) > 8 {
		col = "Imei"
	}
	conf.Db.Exec(
		"UPDATE tab_charging_gun LEFT JOIN tab_product ON pid = product_id SET meter_value = ? WHERE tab_product."+col+" = ? AND direction = ?",
		meter, nid, direction,
	)
}

// currentSeasonPriceTimeID 查询当前季节时段 id（四轮车 tab_order_electronic 用季节表）。
func currentSeasonPriceTimeID() int32 {
	var id int32
	conf.Db.Raw("SELECT id FROM tab_price_time_" + CurrentSeason() + " WHERE start_time < CURTIME() AND over_time >= CURTIME() LIMIT 1").Row().Scan(&id)
	return id
}

// AddOrderElectronic 新增订单充电时段记录（start_value = over_value = meter）。
func AddOrderElectronic(orderID string, meter int32) {
	if meter < 0 {
		return
	}
	pid := currentSeasonPriceTimeID()
	var existing model.TabOrderElectronic
	if err := conf.Db.Where("order_id = ? AND price_time_id = ?", orderID, pid).First(&existing).Error; err == nil {
		return // 当前时段已存在记录，跳过
	}
	conf.Db.Create(&model.TabOrderElectronic{
		OrderID:     orderID,
		StartValue:  decimal.NewFromInt32(meter),
		OverValue:   decimal.NewFromInt32(meter),
		PriceTimeID: pid,
	})
}

// SaveOrUpdateOrderElectronicMeter 保存/更新订单电表表底（对齐 Java saveOrUpdateMeterValue）。
func SaveOrUpdateOrderElectronicMeter(orderID string, meter int32) {
	if meter < 0 {
		return
	}
	var last model.TabOrderElectronic
	if err := conf.Db.Where("order_id = ?", orderID).Order("create_time desc").Limit(1).First(&last).Error; err == nil {
		if meter < int32(last.OverValue.IntPart()) {
			return // 电表值异常（新值小于已有最大值），跳过
		}
		pid := currentSeasonPriceTimeID()
		if last.PriceTimeID == pid {
			conf.Db.Model(&model.TabOrderElectronic{}).Where("order_id = ? AND price_time_id = ?", orderID, pid).Update("over_value", meter)
		} else {
			conf.Db.Create(&model.TabOrderElectronic{OrderID: orderID, StartValue: last.OverValue, OverValue: decimal.NewFromInt32(meter), PriceTimeID: pid})
		}
		return
	}
	AddOrderElectronic(orderID, meter)
}

// UpdateOrderOverTime 更新订单还枪/结束时间。
func UpdateOrderOverTime(orderID string) {
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("over_time", time.Now())
}

// UpdateOrderChargingTime 按 begin_time 计算并更新充电时长（varchar HH:MM:SS）。
func UpdateOrderChargingTime(orderID string) {
	order, err := FindOrderByOrderid(orderID)
	if err != nil || order.BeginTime == nil || order.BeginTime.IsZero() {
		return
	}
	chargingTime := time.Time{}.Add(time.Since(*order.BeginTime)).Format("15:04:05")
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("charging_time", chargingTime)
}

// UpdateOrderBeginTime 更新订单开始时间（拔枪后无车位锁的驿享充电用）。
func UpdateOrderBeginTime(orderID string) {
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("begin_time", time.Now())
}

// UpdateOrderGunDisconnectTime 更新订单枪车断开时间。
func UpdateOrderGunDisconnectTime(orderID string) {
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("gun_disconnect_time", time.Now())
}

// InsertChargingState 新增紧急按钮按下状态。
func InsertChargingState(s *model.ChargingStateTbl) {
	conf.Db.Create(s)
}

// ChargingStatePressedExists 判断是否已存在「按下」的紧急按钮状态。
func ChargingStatePressedExists(productID, direction string) bool {
	var n int64
	conf.Db.Model(&model.ChargingStateTbl{}).
		Where("product_id = ? AND direction = ? AND emergency_button_state = ?", productID, direction, "Pressed").
		Count(&n)
	return n > 0
}

// DeleteChargingState 删除紧急按钮按下状态（复位），返回受影响行数。
func DeleteChargingState(productID, direction string) int64 {
	res := conf.Db.Where("product_id = ? AND direction = ? AND emergency_button_state = ?", productID, direction, "Pressed").
		Delete(&model.ChargingStateTbl{})
	return res.RowsAffected
}

// FindPrivateUseOrderByDevice 按设备标识查询私桩/邻享「使用中」订单。
// 对齐 Java OrderService.getPrivateUseByNid（经 private_place_tbl 关联；4G 设备按 Imei，LoRa 按 nid）。
func FindPrivateUseOrderByDevice(deviceID string) *model.OrderTbl {
	var order model.OrderTbl
	err := conf.Db.Raw(`
		SELECT o.* FROM order_tbl o
		LEFT JOIN private_place_tbl s ON o.spaces_id = s.id
		LEFT JOIN tab_charging_gun g ON s.charging_gun_id = g.id
		LEFT JOIN tab_product gp ON gp.pid = g.product_id
		WHERE o.state = '使用中' AND (gp.nid = ? OR gp.Imei = ?)
		ORDER BY o.reserve_time DESC LIMIT 1`, deviceID, deviceID).Scan(&order).Error
	if err != nil || order.Orderid == "" {
		return nil
	}
	return &order
}

// FindPrivateReturnGunOrderByDevice 按设备标识查询私桩/邻享「待还枪」订单。
// 对齐 Java OrderService.getPrivateReturnGunByNid（经 private_place_tbl 关联；4G 设备按 Imei，LoRa 按 nid）。
func FindPrivateReturnGunOrderByDevice(deviceID string) *model.OrderTbl {
	var order model.OrderTbl
	err := conf.Db.Raw(`
		SELECT o.* FROM order_tbl o
		LEFT JOIN private_place_tbl s ON o.spaces_id = s.id
		LEFT JOIN tab_charging_gun g ON s.charging_gun_id = g.id
		LEFT JOIN tab_product gp ON gp.pid = g.product_id
		WHERE o.state = '待还枪' AND (gp.nid = ? OR gp.Imei = ?)
		ORDER BY o.reserve_time DESC LIMIT 1`, deviceID, deviceID).Scan(&order).Error
	if err != nil || order.Orderid == "" {
		return nil
	}
	return &order
}

// GaugeFullStep 二轮车充满检测：功率低于阈值则累计 count，返回 (新 count, 是否达到关闭条件)。
// 对齐 Java MessageListeningServiceImpl.GaugeFull（功率 < 15W 计 1 次，每满 3 次触发关闭）。
func GaugeFullStep(orderID string) (int32, bool) {
	var rec model.ReOrderElectronicTbl
	if err := conf.Db.Where("order_id = ?", orderID).First(&rec).Error; err != nil {
		return 0, false
	}
	power, err := strconv.ParseFloat(rec.Power, 64)
	if err != nil || power >= 15 {
		return 0, false
	}
	newCount := rec.Count + 1
	conf.Db.Model(&model.ReOrderElectronicTbl{}).Where("order_id = ?", orderID).Update("count", newCount)
	return newCount, newCount > 0 && newCount%3 == 0
}
