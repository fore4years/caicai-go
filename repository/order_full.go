package repository

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// 本文件对应 handler/order_full.go（OrderController 的查询/下单/取消/结算/二轮车计费）
// 的内联 SQL，统一抽离为命名函数。
// ============================================================================

// ---------------------------- 订单查询 ----------------------------

// FirstUnfinishedOrderByOpenid 查询用户未完成订单（state 非已完成/已取消），不存在返回 nil。
func FirstUnfinishedOrderByOpenid(openid string) *model.OrderTbl {
	var o model.OrderTbl
	if err := conf.Db.Where("openid = ? AND state <> ? AND state <> ?", openid, OrderStateFinished, OrderStateRevoked).
		First(&o).Error; err != nil {
		return nil
	}
	return &o
}

// UnfinishedNonNeighborOrdersBySpacesID 按车位查询未完成订单（排除邻享充电）。
func UnfinishedNonNeighborOrdersBySpacesID(spacesID int32) []model.OrderTbl {
	var list []model.OrderTbl
	conf.Db.Where("spaces_id = ? AND state <> ? AND state <> ? AND consumption_type <> ?",
		spacesID, OrderStateRevoked, OrderStateFinished, ConsumeTypeNeighbor).Find(&list)
	return list
}

// UnfinishedOrdersBySpacesID 按车位查询未完成订单（不区分消费类型）。
func UnfinishedOrdersBySpacesID(spacesID int32) []model.OrderTbl {
	var list []model.OrderTbl
	conf.Db.Where("spaces_id = ? AND state <> ? AND state <> ?", spacesID, OrderStateRevoked, OrderStateFinished).Find(&list)
	return list
}

// LatestOrdersByOpenid 查询用户最近订单（预约时间倒序）。
func LatestOrdersByOpenid(openid string, limit int) []model.OrderTbl {
	var orders []model.OrderTbl
	conf.Db.Where("openid = ?", openid).Order("reserve_time desc").Limit(limit).Find(&orders)
	return orders
}

// PageOrdersByOpenid 按 openid 分页查询订单（预约时间倒序）。
func PageOrdersByOpenid(openid string, offset, limit int) (int64, []model.OrderTbl) {
	var total int64
	conf.Db.Model(&model.OrderTbl{}).Where("openid = ?", openid).Count(&total)
	var orders []model.OrderTbl
	conf.Db.Where("openid = ?", openid).Order("reserve_time desc").Offset(offset).Limit(limit).Find(&orders)
	return total, orders
}

// OrdersByOpenidAndState 按 openid + 状态查询订单。
func OrdersByOpenidAndState(openid, state string) []model.OrderTbl {
	var orders []model.OrderTbl
	conf.Db.Where("openid = ? AND state = ?", openid, state).Find(&orders)
	return orders
}

// OngoingOrdersByOpenid 按 openid 查询进行中订单（使用中/已预约/预约中）。
func OngoingOrdersByOpenid(openid string) []model.OrderTbl {
	var orders []model.OrderTbl
	conf.Db.Where("openid = ? AND state IN ('使用中','已预约','预约中')", openid).Find(&orders)
	return orders
}

// PageOrdersByOpenidAndConsumption 按 openid + 消费类型分页查询订单。
func PageOrdersByOpenidAndConsumption(openid, consumption string, offset, limit int) (int64, []model.OrderTbl) {
	var total int64
	conf.Db.Model(&model.OrderTbl{}).Where("openid = ? AND consumption_type = ?", openid, consumption).Count(&total)
	var orders []model.OrderTbl
	conf.Db.Where("openid = ? AND consumption_type = ?", openid, consumption).Order("reserve_time desc").Offset(offset).Limit(limit).Find(&orders)
	return total, orders
}

// WaitReturnGunOrderByPrivatePlaceID 查询私桩车位的待还枪订单。
func WaitReturnGunOrderByPrivatePlaceID(id string) (*model.OrderTbl, error) {
	var order model.OrderTbl
	err := conf.Db.Raw(
		`SELECT o.* FROM order_tbl o LEFT JOIN private_place_tbl p ON p.id = o.spaces_id WHERE p.id = ? AND o.state = '待还枪'`,
		id,
	).Scan(&order).Error
	if err != nil || order.Orderid == "" {
		return nil, err
	}
	return &order, nil
}

// ActiveCaiParkingOrdersByLockid 查询锁下进行中的踩踩停车订单。
func ActiveCaiParkingOrdersByLockid(lockid string) []model.OrderTbl {
	var orders []model.OrderTbl
	conf.Db.Where("lockid = ? AND consumption_type = ? AND state IN ('使用中','已预约','预约中')", lockid, ConsumeTypeCaiParking).Find(&orders)
	return orders
}

// UnfinishedNeighborOrderBySpacesID 查询车位下的邻享充电未完成订单。
func UnfinishedNeighborOrderBySpacesID(spacesID int32) (*model.OrderTbl, error) {
	var existing model.OrderTbl
	if err := conf.Db.Where("consumption_type = ? AND spaces_id = ? AND state IN ('进行中','使用中','待还枪')",
		ConsumeTypeNeighbor, spacesID).First(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

// ---------------------------- 订单写入 ----------------------------

// CreateOrder 新增订单。
func CreateOrder(order *model.OrderTbl) error {
	return conf.Db.Create(order).Error
}

// CreateConfig 新增订单配置记录（charge_gun=0）。
func CreateConfig(orderid string) error {
	return conf.Db.Create(&model.ConfigTbl{Orderid: orderid, ChargeGun: 0}).Error
}

// UpdateOrderState 更新订单状态，返回受影响行数。
func UpdateOrderState(orderid, state string) int64 {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderid).Update("state", state)
	return res.RowsAffected
}

// UpdateOrderFieldsByState 按 orderid + 当前状态更新订单字段，返回受影响行数。
func UpdateOrderFieldsByState(orderid, state string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderid, state).Updates(updates)
	return res.RowsAffected
}

// UpdateOrderFieldsByStates 按 orderid + 当前状态集合更新订单字段，返回受影响行数。
func UpdateOrderFieldsByStates(orderid string, states []string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state IN ?", orderid, states).Updates(updates)
	return res.RowsAffected
}

// UpdateOrderBasicConsumption 更新订单基本消费金额，返回受影响行数。
func UpdateOrderBasicConsumption(orderid string, amount decimal.Decimal) int64 {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderid).Update("basic_consumption", amount)
	return res.RowsAffected
}

// UpdateDriverUseReserveNumber 更新用户剩余预约次数，返回受影响行数。
func UpdateDriverUseReserveNumber(openid string, n int32) int64 {
	res := conf.Db.Model(&model.DriveruseTbl{}).Where("openid = ?", openid).Update("reserveNumber", n)
	return res.RowsAffected
}

// CountOrdersBySpacesIDState 统计车位某状态的订单数。
func CountOrdersBySpacesIDState(spacesID int32, state string) int64 {
	var cnt int64
	conf.Db.Model(&model.OrderTbl{}).Where("spaces_id = ? AND state = ?", spacesID, state).Count(&cnt)
	return cnt
}

// ---------------------------- 插座 / 二轮车电量 ----------------------------

// CountReOrderElectronicsByOrderID 统计订单二轮车电量记录数。
func CountReOrderElectronicsByOrderID(orderID string) int64 {
	var count int64
	conf.Db.Model(&model.ReOrderElectronicTbl{}).Where("order_id = ?", orderID).Count(&count)
	return count
}

// InsertReOrderElectronic 新增二轮车订单电量记录。
func InsertReOrderElectronic(orderID string, startValue decimal.Decimal) error {
	return conf.Db.Exec(
		"INSERT INTO re_order_electronic_tbl (order_id, start_value, over_value, price_time_id) VALUES (?, ?, ?, (SELECT id FROM tab_price_time WHERE start_time < CURTIME() AND over_time >= CURTIME() LIMIT 1))",
		orderID, startValue, startValue,
	).Error
}

// ActivateReceptacle 将空闲插座置为使用中，返回受影响行数。
func ActivateReceptacle(id int32) int64 {
	res := conf.Db.Model(&model.ReceptacleTbl{}).Where("id = ? AND status = 0", id).Update("status", 1)
	return res.RowsAffected
}

// ---------------------------- 二轮车计费 ----------------------------

// ReOrderPowerValue 查询订单期间充电量（MAX(over_value)-MIN(start_value)）。
func ReOrderPowerValue(orderID string) decimal.Decimal {
	var v decimal.Decimal
	conf.Db.Raw("SELECT COALESCE((MAX(over_value)-MIN(start_value)),0) FROM re_order_electronic_tbl WHERE order_id = ?", orderID).Row().Scan(&v)
	return v
}

// ReOrderPowerNoServiceFees 查询订单电费（不含服务费）。
func ReOrderPowerNoServiceFees(orderID string) decimal.Decimal {
	var v decimal.Decimal
	conf.Db.Raw(`SELECT COALESCE(SUM((over_value - start_value) * COALESCE(charging_price_tbl.total, 0)),0)
		FROM re_order_electronic_tbl
		LEFT JOIN tab_price_time ON tab_price_time.id = re_order_electronic_tbl.price_time_id
		LEFT JOIN charging_price_tbl ON charging_price_tbl.id = tab_price_time.price_id
		WHERE re_order_electronic_tbl.order_id = ?`, orderID).Row().Scan(&v)
	return v
}

// ReOrderPowerMaxPower 查询订单期间最大功率。
func ReOrderPowerMaxPower(orderID string) float64 {
	var v float64
	conf.Db.Raw("SELECT COALESCE(max_power,0) FROM re_order_electronic_tbl WHERE order_id = ? LIMIT 1", orderID).Row().Scan(&v)
	return v
}

// ServiceFeeByPowerRange 按功率区间查询服务费率。
func ServiceFeeByPowerRange(maxPower float64) decimal.Decimal {
	if maxPower <= 0 {
		return MaxServiceFee()
	}
	var v decimal.Decimal
	if conf.Db.Raw("SELECT service FROM charging_price_tbl WHERE ? >= min_power AND ? <= max_power LIMIT 1", maxPower, maxPower).Row().Scan(&v) != nil {
		return MaxServiceFee()
	}
	return v
}

// MaxServiceFee 返回充电价格表中的最高服务费率（表空默认 0.4）。
func MaxServiceFee() decimal.Decimal {
	var v decimal.Decimal
	if conf.Db.Raw("SELECT COALESCE(MAX(service), 0.4) FROM charging_price_tbl").Row().Scan(&v) != nil {
		return decimal.RequireFromString("0.4")
	}
	return v
}

// UserIsFreeServiceByOrder 判断订单用户是否享受免服务费。
func UserIsFreeServiceByOrder(orderID string) bool {
	order, err := FindOrderByOrderid(orderID)
	if err != nil {
		return false
	}
	var users []model.FreeChargingUsersTbl
	conf.Db.Raw(`SELECT fcu.* FROM free_charging_users_tbl fcu
		LEFT JOIN order_tbl o ON o.openid = fcu.openid
		WHERE o.orderid = ? AND fcu.status = 1`, orderID).Scan(&users)
	for _, u := range users {
		var productID string
		conf.Db.Raw("SELECT station.product_id FROM charging_station_management_tbl station WHERE station.id = ?", u.ChargingStationID).Scan(&productID)
		if order.Pid == productID {
			return true
		}
	}
	return false
}

// UpdateChargingDegreeForOrder 计算并更新订单充电度数。
func UpdateChargingDegreeForOrder(orderID string) {
	var degree decimal.Decimal
	conf.Db.Raw("SELECT COALESCE(SUM(over_value - start_value),0)/100 FROM tab_order_electronic WHERE order_id = ?", orderID).
		Row().Scan(&degree)
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("charging_degree", degree)
}

// ---------------------------- 二轮车结算事务 ----------------------------

// SettleTwiceOrderTx 二轮车订单结算落库事务：写入计费记录、套餐时长、余额/微信扣款、插座与订单状态。
// completeWx 由调用方传入（微信支付分完结），以保留原「事务内调用」的时序。
func SettleTwiceOrderTx(orderID, openid string, powerRate, serviceRate, fee, meterValue decimal.Decimal,
	newPackageDuration *int32, packageOrderNo string, receptacleID int32, chargingTime string, now time.Time,
	wxCompleted bool, completeWx func(orderID string, amountFen int)) error {
	return conf.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.OrderTwiceTbl{
			Orderid:     orderID,
			ServiceRate: serviceRate,
			PowerRate:   powerRate,
		}).Error; err != nil {
			return err
		}
		if newPackageDuration != nil {
			if err := tx.Model(&model.PackageRecordTbl{}).Where("order_no = ?", packageOrderNo).
				Update("duration", *newPackageDuration).Error; err != nil {
				return err
			}
		}

		if wxCompleted {
			completeWx(orderID, int(fee.Mul(decimal.NewFromInt(100)).IntPart()))
		} else if fee.IsPositive() {
			ok, _, cerr := ConsumeBalanceTx(tx, openid, fee)
			if cerr != nil {
				return cerr
			}
			if !ok {
				return errors.New("余额不足")
			}
			if err := tx.Create(&model.TabBalanceRecord{Openid: openid, Type: "充电消费", Amount: fee}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Exec("UPDATE user_tbl SET balans = balans + freeze_balans, freeze_balans = 0 WHERE openid = ?", openid).Error; err != nil {
				return err
			}
		}

		if receptacleID != 0 {
			if err := tx.Model(&model.ReceptacleTbl{}).Where("id = ?", receptacleID).Update("status", 0).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("over_time", now).Error; err != nil {
			return err
		}
		if chargingTime != "" {
			if err := tx.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("charging_time", chargingTime).Error; err != nil {
				return err
			}
		}
		return tx.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Updates(map[string]interface{}{
			"state":             OrderStateFinished,
			"close_time":        now,
			"charging_degree":   meterValue,
			"basic_consumption": fee.Round(2),
		}).Error
	})
}
