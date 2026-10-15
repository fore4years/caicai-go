package repository

import (
	"fmt"
	"strconv"

	"caicai-go/conf"
	"caicai-go/model"
)

// PlaceWithLock 车位及其锁 id。
type PlaceWithLock struct {
	model.PlaceTbl
	LockID string `gorm:"column:lockid"`
}

// NextPlaceSpacesCode code + 5 位递增编号。
func NextPlaceSpacesCode(code string) string {
	if code == "" {
		return ""
	}
	var maxCode string
	conf.Db.Raw("SELECT MAX(placeid) FROM place_tbl WHERE placeid LIKE ?", code+"%").Scan(&maxCode)
	if maxCode == "" {
		maxCode = "0"
	}
	remainder := maxCode
	if len(maxCode) >= len(code) && maxCode[:len(code)] == code {
		remainder = maxCode[len(code):]
	}
	n, err := strconv.Atoi(remainder)
	if err != nil {
		return ""
	}
	return code + fmt.Sprintf("%05d", n+1)
}

// PlacesWithLockByOwnerid 按 ownerid 查询车位及锁 id。
func PlacesWithLockByOwnerid(ownerid string) []PlaceWithLock {
	var rows []PlaceWithLock
	conf.Db.Raw(
		`SELECT p.*, l.lockid FROM place_tbl p
		 LEFT JOIN lock_tbl l ON p.placeid = l.placeid
		 WHERE ownerid = ? ORDER BY placeid ASC`, ownerid,
	).Scan(&rows)
	return rows
}

// ChargeMessagesByPlaceid 按 placeid 查询充电桩消息。
func ChargeMessagesByPlaceid(placeid string) []model.ChargeMessageTbl {
	var cms []model.ChargeMessageTbl
	conf.Db.Where("placeid = ?", placeid).Find(&cms)
	return cms
}

// OrdersByOwnerid 按 ownerid 查询已完成订单（begin_time 倒序）。
func OrdersByOwnerid(ownerid string) []model.OrderTbl {
	var orders []model.OrderTbl
	conf.Db.Raw(
		`SELECT * FROM order_tbl
		 WHERE lockid IN (SELECT lockid FROM lock_tbl WHERE placeid IN (SELECT placeid FROM place_tbl WHERE ownerid = ?))
		   AND state = '已完成'
		 ORDER BY begin_time DESC`, ownerid,
	).Scan(&orders)
	return orders
}

// PlaceapplyExists 判断车位申请是否已存在。
func PlaceapplyExists(name, code, streer, relatedBuilding, carnumber string) bool {
	var existing model.PlaceapplyTbl
	err := conf.Db.Where("name = ? AND code = ? AND streer = ? AND relatedBuilding = ? AND carnumber = ?",
		name, code, streer, relatedBuilding, carnumber).First(&existing).Error
	return err == nil
}

// UpdatePlaceFields 更新车位字段，返回受影响行数。
func UpdatePlaceFields(placeid string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Updates(updates)
	return res.RowsAffected
}

// UpdatePlaceField 更新车位单字段。
func UpdatePlaceField(placeid, field string, value interface{}) {
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Update(field, value)
}

// UpdateParkingSpaceLockID 更新车位锁 id。
func UpdateParkingSpaceLockID(spacesCode, lockID string) {
	conf.Db.Model(&model.TabParkingSpace{}).Where("spaces_code = ?", spacesCode).Update("lock_id", lockID)
}

// UpdatePlaceapplyField 更新车位申请单字段。
func UpdatePlaceapplyField(id, field string, value interface{}) {
	conf.Db.Model(&model.PlaceapplyTbl{}).Where("id = ?", id).Update(field, value)
}

// MessagesByOwnerid 按 ownerid 查询业主消息。
func MessagesByOwnerid(ownerid string) []model.MessageTbl {
	var msgs []model.MessageTbl
	conf.Db.Where("ownerid = ?", ownerid).Find(&msgs)
	return msgs
}
