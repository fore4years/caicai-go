package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/objects"
)

// GatewayByMac 按 mac 查询网关，未找到返回 nil。
func GatewayByMac(mac string) *model.TabGateway {
	var g model.TabGateway
	if conf.Db.Where("mac = ?", mac).First(&g).Error != nil {
		return nil
	}
	return &g
}

// InsertGateway 注册网关（mac/iccid）。
func InsertGateway(mac, iccid string) error {
	return conf.Db.Exec("INSERT INTO tab_gateway(mac, iccid) VALUES (?, ?)", mac, iccid).Error
}

// LockProductGatewayByOrder 按订单查询锁及其产品/网关信息（对应 Java LockService.getByOrder + product + gateway）。
func LockProductGatewayByOrder(orderID string) (model.LockTbl, *model.TabProduct, *model.TabGateway, error) {
	var lock model.LockTbl
	if err := conf.Db.Raw(
		"SELECT l.* FROM order_tbl o LEFT JOIN tab_parking_spaces ps ON o.spaces_id = ps.id LEFT JOIN lock_tbl l ON ps.lock_id = l.lockid WHERE o.orderid = ?",
		orderID,
	).Scan(&lock).Error; err != nil {
		return lock, nil, nil, err
	}
	if lock.Lockid == "" {
		return lock, nil, nil, nil
	}
	product, err := objects.TabProduct.WithContext(context.Background()).Where(objects.TabProduct.Pid.Eq(lock.ProductID)).First()
	if err != nil {
		return lock, nil, nil, err
	}
	gateway, err := objects.TabGateway.WithContext(context.Background()).Where(objects.TabGateway.ID.Eq(product.GatewayID)).First()
	if err != nil {
		return lock, nil, nil, err
	}
	return lock, product, gateway, nil
}

// CountActiveOrdersByLock 统计锁上其它进行中/使用中的订单数。
func CountActiveOrdersByLock(lockid, excludeOrderID string) int64 {
	var count int64
	conf.Db.Model(&model.OrderTbl{}).
		Where("lockid = ? AND orderid <> ? AND state IN ('使用中','进行中')", lockid, excludeOrderID).
		Count(&count)
	return count
}

// UpdatePlaceState 更新车位状态。
func UpdatePlaceState(placeid, state string) {
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Update("state", state)
}

// StartOrder 将订单置为使用中并记录开始时间。
func StartOrder(orderID string, beginTime time.Time) {
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Updates(map[string]interface{}{
		"state":      "使用中",
		"begin_time": beginTime,
	})
}

// FindUseOrderByDevice 按设备标识（nid）查询「使用中」订单。
func FindUseOrderByDevice(deviceID string) *model.OrderTbl {
	var order model.OrderTbl
	err := conf.Db.Raw(`
		SELECT o.* FROM order_tbl o
		LEFT JOIN tab_parking_spaces s ON o.spaces_id = s.id
		LEFT JOIN tab_charging_gun g ON s.charging_gun_id = g.id
		LEFT JOIN tab_product gp ON gp.pid = g.product_id
		LEFT JOIN lock_tbl l ON s.lock_id = l.lockid
		LEFT JOIN tab_product lp ON lp.pid = l.product_id
		WHERE o.state = '使用中'
		  AND (gp.nid = ? OR gp.Imei = ? OR lp.nid = ? OR lp.Imei = ?)
		ORDER BY o.reserve_time DESC
		LIMIT 1`, deviceID, deviceID, deviceID, deviceID).Scan(&order).Error
	if err != nil || order.Orderid == "" {
		logger.Mylog.Debug().Msgf("未找到设备对应的使用中订单, device: %s", deviceID)
		return nil
	}
	return &order
}

// FindYixiangUseOrderByNidDirection 按设备标识 + 充电枪方向查询驿享充电「使用中」订单。
// 对齐 Java OrderService.getUseByNidAndDirection（nid+方向关联，且限定消费类型为驿享充电）。
func FindYixiangUseOrderByNidDirection(deviceID, direction string) *model.OrderTbl {
	var order model.OrderTbl
	err := conf.Db.Raw(`
		SELECT o.* FROM order_tbl o
		LEFT JOIN tab_parking_spaces s ON o.spaces_id = s.id
		LEFT JOIN tab_charging_gun g ON s.charging_gun_id = g.id
		LEFT JOIN tab_product gp ON gp.pid = g.product_id
		WHERE o.state = '使用中'
		  AND o.consumption_type = '驿享充电'
		  AND g.direction = ?
		  AND (gp.nid = ? OR gp.Imei = ?)
		ORDER BY o.reserve_time DESC
		LIMIT 1`, direction, deviceID, deviceID).Scan(&order).Error
	if err != nil || order.Orderid == "" {
		logger.Mylog.Debug().Msgf("未找到设备对应的驿享充电使用中订单, device: %s, direction: %s", deviceID, direction)
		return nil
	}
	return &order
}

// UpdateOrderFullTime 更新订单充满时间。
func UpdateOrderFullTime(orderID string) {
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("full_time", time.Now())
}

// UpdateOrderChargeGunPull 更新订单插/还枪状态。
func UpdateOrderChargeGunPull(orderID string, pulled int) {
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("charge_gun_pull", pulled)
}

// ProductByDeviceID 按设备标识查询产品：长度>8 视为 4G IMEI，否则视为 LoRa nid。
func ProductByDeviceID(deviceID string) *model.TabProduct {
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

// UpsertReceptaclePower 更新插座电量（不存在则新增）。
func UpsertReceptaclePower(pid, channel string, meterValue decimal.Decimal) {
	cid, _ := strconv.Atoi(channel)
	var existing model.ReceptaclePowerTbl
	if err := conf.Db.Where("pid = ? AND receptacle_id = ?", pid, channel).First(&existing).Error; err != nil {
		conf.Db.Create(&model.ReceptaclePowerTbl{Pid: pid, ReceptacleID: int32(cid), MeterValue: meterValue})
		return
	}
	conf.Db.Model(&model.ReceptaclePowerTbl{}).Where("pid = ? AND receptacle_id = ?", pid, channel).Update("meter_value", meterValue)
}

// FindTwiceOrderByPidAndChannel 查询使用中的二轮车订单。
func FindTwiceOrderByPidAndChannel(pid, channel string) *model.OrderTbl {
	var order model.OrderTbl
	if err := conf.Db.Where("pid = ? AND lockid = ? AND state = ?", pid, channel, "使用中").First(&order).Error; err != nil {
		return nil
	}
	return &order
}

// UpdateReOrderPower 更新订单充电电量（累计表底，二进制协议）。
func UpdateReOrderPower(orderID string, meterValue decimal.Decimal, power string) {
	newPower, _ := decimal.NewFromString(power)
	conf.Db.Exec(
		"UPDATE re_order_electronic_tbl SET over_value = ?, power = ?, max_power = GREATEST(COALESCE(max_power,0), ?), price_time_id = (SELECT id FROM tab_price_time WHERE start_time < CURTIME() AND over_time >= CURTIME() LIMIT 1) WHERE order_id = ?",
		meterValue, power, newPower, orderID,
	)
}

// SetReOrderPower 覆盖式更新订单充电电量（MQTT 上报电量直接覆盖）。
func SetReOrderPower(orderID string, energyKwh, powerW decimal.Decimal) {
	conf.Db.Exec(
		"UPDATE re_order_electronic_tbl SET over_value = ?, power = ?, max_power = GREATEST(COALESCE(max_power,0), ?), price_time_id = (SELECT id FROM tab_price_time WHERE start_time < CURTIME() AND over_time >= CURTIME() LIMIT 1) WHERE order_id = ?",
		energyKwh, powerW.StringFixed(2), powerW, orderID,
	)
}
