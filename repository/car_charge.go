package repository

import (
	"time"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// 本文件对应 handler/car_charge.go 中 /wx 前缀的两个旧控制器（CarDetailsCon、
// ChargeListCon）的内联 SQL，统一抽离为命名函数。
// ============================================================================

// ---------------------------- CarDetailsController ----------------------------

// UnfinishedOrdersByOpenid 查询用户未完成订单（state 非已完成/已取消）。
func UnfinishedOrdersByOpenid(openid string) []model.OrderTbl {
	var list []model.OrderTbl
	conf.Db.Where("openid = ? AND state <> ? AND state <> ?", openid, "已完成", "已取消").Find(&list)
	return list
}

// ActiveOrdersByLockid 查询该锁进行中的订单（使用中/已预约/预约中）。
func ActiveOrdersByLockid(lockid string) []model.OrderTbl {
	var orders []model.OrderTbl
	conf.Db.Where("lockid = ? AND state IN (?)", lockid, []string{"使用中", "已预约", "预约中"}).Find(&orders)
	return orders
}

// InsertUsermessage 新增用户消息（未读）。
func InsertUsermessage(openid, message string) {
	conf.Db.Create(&model.UsermessageTbl{
		Openid:  openid,
		Message: message,
		MsgTime: time.Now().Format("2006-01-02 15:04:05"),
		Status:  "未读",
	})
}

// UpdatePlaceStateByLockidJoin 通过 lockid 关联 placeid 更新车位状态（LEFT JOIN）。
func UpdatePlaceStateByLockidJoin(lockid, state string) {
	conf.Db.Exec(
		"UPDATE place_tbl LEFT JOIN lock_tbl ON place_tbl.placeid = lock_tbl.placeid SET place_tbl.state = ? WHERE lockid = ?",
		state, lockid,
	)
}

// ClearSaveyyTime 清空预约记录的退出时间与锁。
func ClearSaveyyTime(id string) {
	conf.Db.Exec("UPDATE saveyytime_tbl SET haveTime = NULL, exitTime = NULL, lockid = NULL WHERE id = ?", id)
}

// UpdateSaveyyTime 记录预约退出时间，返回受影响行数。
func UpdateSaveyyTime(havetime, lockid, id string) int64 {
	res := conf.Db.Exec(
		"UPDATE saveyytime_tbl SET haveTime = ?, exitTime = ?, lockid = ? WHERE id = ?",
		havetime, time.Now().Format("2006-01-02 15:04:05"), lockid, id,
	)
	return res.RowsAffected
}

// SaveyyTimeByOpenidLockidID 按 openid+lockid+id 查询预约记录。
func SaveyyTimeByOpenidLockidID(openid, lockid, id string) (*model.SaveyytimeTbl, error) {
	var st model.SaveyytimeTbl
	if err := conf.Db.Where("openid = ? AND lockid = ? AND id = ?", openid, lockid, id).First(&st).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// InsertSaveyyTime 新增预约记录。
func InsertSaveyyTime(id, openid, lockid string) {
	conf.Db.Exec("INSERT INTO saveyytime_tbl (id, openid, lockid) VALUES (?, ?, ?)", id, openid, lockid)
}

// UsercarsByOpenid 查询用户车牌列表。
func UsercarsByOpenid(openid string) []model.UsercarTbl {
	var rows []model.UsercarTbl
	conf.Db.Where("openid = ?", openid).Find(&rows)
	return rows
}

// UsercarByPlateNum 按车牌查询未删除车辆。
func UsercarByPlateNum(plateNum string) (*model.UsercarTbl, error) {
	var u model.UsercarTbl
	if err := conf.Db.Where("plate_num = ? AND isDelete = 0", plateNum).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// InsertUsercar 新增车牌。
func InsertUsercar(openid, plateNum string) {
	conf.Db.Create(&model.UsercarTbl{Openid: openid, PlateNum: plateNum, State: "未认证"})
}

// DeleteUsercarByPlateNum 按车牌删除车辆，返回受影响行数。
func DeleteUsercarByPlateNum(plateNum string) int64 {
	res := conf.Db.Where("plate_num = ?", plateNum).Delete(&model.UsercarTbl{})
	return res.RowsAffected
}

// ---------------------------- ChargeListController ----------------------------

// NearbyChargePlaces 查询附近可充电车位（按经纬度距离排序，最多 100 条）。
func NearbyChargePlaces(longitude, latitude string) []model.PlaceTbl {
	var places []model.PlaceTbl
	conf.Db.Raw(
		`SELECT p.* FROM place_tbl p
		 INNER JOIN (SELECT placeid FROM place_tbl WHERE state = '可使用' AND ischarge = 1) t ON p.placeid = t.placeid
		 WHERE p.longitude IS NOT NULL AND p.latitude IS NOT NULL
		 ORDER BY (POWER(MOD(ABS(longitude - ?), 360), 2) + POWER(ABS(latitude - ?), 2))
		 LIMIT 100`,
		longitude, latitude,
	).Scan(&places)
	return places
}

// LockidByPlaceid 按 placeid 查询锁 id。
func LockidByPlaceid(placeid string) string {
	var lockid string
	conf.Db.Raw("SELECT lockid FROM lock_tbl WHERE placeid = ?", placeid).Row().Scan(&lockid)
	return lockid
}

// AllLocksDetail 查询所有车位锁（完整记录）。
func AllLocksDetail() []model.LockTbl {
	var rows []model.LockTbl
	conf.Db.Find(&rows)
	return rows
}

// AllChargeMessages 查询所有充电桩信息。
func AllChargeMessages() []model.ChargeMessageTbl {
	var rows []model.ChargeMessageTbl
	conf.Db.Find(&rows)
	return rows
}

// ChargeMessagesByChargeid 按 chargeid 查询充电桩。
func ChargeMessagesByChargeid(chargeid string) []model.ChargeMessageTbl {
	var rows []model.ChargeMessageTbl
	conf.Db.Where("chargeid = ?", chargeid).Find(&rows)
	return rows
}

// LocksByLockid 按 lockid 查询车位锁。
func LocksByLockid(lockid string) []model.LockTbl {
	var rows []model.LockTbl
	conf.Db.Where("lockid = ?", lockid).Find(&rows)
	return rows
}

// PlacesWithLocks 查询已安装车位锁的车位。
func PlacesWithLocks() []model.PlaceTbl {
	var places []model.PlaceTbl
	conf.Db.Raw(
		"SELECT place_tbl.* FROM place_tbl INNER JOIN lock_tbl ON place_tbl.placeid = lock_tbl.placeid",
	).Scan(&places)
	return places
}

// AllPlaces 查询所有车位。
func AllPlaces() []model.PlaceTbl {
	var places []model.PlaceTbl
	conf.Db.Find(&places)
	return places
}

// UpdateLockInstallTime 更新锁安装时间与安装人员，返回受影响行数。
func UpdateLockInstallTime(installTime, lockid, name string) int64 {
	res := conf.Db.Exec("UPDATE lock_tbl SET install_time = ?, name = ? WHERE lockid = ?", installTime, name, lockid)
	return res.RowsAffected
}

// UpdatePlaceIscarlock 更新车位是否安装锁，返回受影响行数。
func UpdatePlaceIscarlock(iscarlock int, placeid string) int64 {
	res := conf.Db.Exec("UPDATE place_tbl SET iscarlock = ? WHERE placeid = ?", iscarlock, placeid)
	return res.RowsAffected
}

// FaultLockPlaceids 查询故障锁的 placeid 列表。
func FaultLockPlaceids() []string {
	var rows []struct {
		Placeid string
	}
	conf.Db.Raw("SELECT l.placeid FROM lock_tbl l JOIN fault_tbl f ON l.lockid = f.lockmac").Scan(&rows)
	list := make([]string, 0, len(rows))
	for _, r := range rows {
		list = append(list, r.Placeid)
	}
	return list
}

// LocksByProductID 按产品 id 查询车位锁。
func LocksByProductID(productID string) []model.LockTbl {
	var rows []model.LockTbl
	conf.Db.Where("product_id = ?", productID).Find(&rows)
	return rows
}

// ChargeMessagesByProductIDDirection 按产品 id 与方向查询充电桩。
func ChargeMessagesByProductIDDirection(productID, direction string) []model.ChargeMessageTbl {
	var rows []model.ChargeMessageTbl
	conf.Db.Where("ProductID = ? AND direction = ?", productID, direction).Find(&rows)
	return rows
}

// UpdateChargeOrderFields 按 orderid 更新充电订单字段，返回受影响行数。
func UpdateChargeOrderFields(orderid string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.ChargeOrderTbl{}).Where("orderid = ?", orderid).Updates(updates)
	return res.RowsAffected
}
