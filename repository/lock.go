package repository

import (
	"time"

	"caicai-go/conf"
	"caicai-go/model"
)

// CreateLockOrder 插入"进行中"订单，返回订单号。
func CreateLockOrder(openid, lockid, orderid string) string {
	now := time.Now()
	conf.Db.Create(&model.OrderTbl{
		Orderid:   orderid,
		Openid:    openid,
		Lockid:    lockid,
		BeginTime: &now,
		State:     OrderStateConduct, // 进行中
	})
	return orderid
}

// UpdateLockOrder 把该锁"预约中"订单更新为"进行中"，返回订单号。
func UpdateLockOrder(lockid string) string {
	var o model.OrderTbl
	if err := conf.Db.Where("lockid = ? AND state = ?", lockid, OrderStateBooking).First(&o).Error; err != nil {
		return ""
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", o.Orderid).Updates(map[string]interface{}{
		"begin_time": time.Now(),
		"state":      OrderStateConduct,
	})
	return o.Orderid
}

// UpdatePlaceByLockid 按 lockid 关联 placeid 更新车位状态。
func UpdatePlaceByLockid(lockid, state string) {
	p := PlaceByLockid(lockid)
	if p != nil {
		conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", p.Placeid).Update("state", state)
	}
}

// CancelYuyue 取消预约的 DB 副作用，返回取消订单的受影响行数。
func CancelYuyue(openid string) int64 {
	// pdao.updateStateByOpenidONUsedOrder(openid, "可使用")
	conf.Db.Exec(`UPDATE place_tbl
		LEFT JOIN lock_tbl lt ON place_tbl.placeid = lt.placeid
		LEFT JOIN tab_parking_spaces ts ON lt.lockid = ts.lock_id
		LEFT JOIN order_tbl ot ON ts.id = ot.spaces_id
		SET place_tbl.state = '可使用'
		WHERE ot.openid = ? AND (ot.state = '预约中' OR ot.state = '已预约')`, openid)
	// dudao.updateReserve(openid)：reserveNumber = reserveNumber - 1
	conf.Db.Exec("UPDATE driveruse_tbl SET reserveNumber = reserveNumber - 1 WHERE openid = ?", openid)
	// odao.cancelyuyue(openid)：订单状态 → 已取消
	return conf.Db.Exec("UPDATE order_tbl SET state = '已取消' WHERE openid = ? AND state = '预约中'", openid).RowsAffected
}

// BookingOrderByLockid 查询该锁的"预约中"订单。
func BookingOrderByLockid(lockid string) (*model.OrderTbl, error) {
	var o model.OrderTbl
	if err := conf.Db.Where("lockid = ? AND state = ?", lockid, OrderStateBooking).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// LockByLockid 按 lockid 查询车位锁。
func LockByLockid(lockid string) (*model.LockTbl, error) {
	var lock model.LockTbl
	if err := conf.Db.Where("lockid = ?", lockid).First(&lock).Error; err != nil {
		return nil, err
	}
	return &lock, nil
}
