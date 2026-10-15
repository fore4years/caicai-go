package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// VideoWatchRecordsByOpenid 查询某用户的视频观看记录。
func VideoWatchRecordsByOpenid(openid string) []model.TabVideoWatchRecord {
	var rows []model.TabVideoWatchRecord
	conf.Db.Where("openid = ?", openid).Find(&rows)
	return rows
}

// CountVideoWatchRecordsByFileID 统计某视频文件的观看次数。
func CountVideoWatchRecordsByFileID(fileID string) int64 {
	var n int64
	conf.Db.Model(&model.TabVideoWatchRecord{}).Where("file_id = ?", fileID).Count(&n)
	return n
}

// ReceptaclePowersByPid 查询某产品的插座电量记录。
func ReceptaclePowersByPid(pid string) []model.ReceptaclePowerTbl {
	var rows []model.ReceptaclePowerTbl
	conf.Db.Where("pid = ?", pid).Find(&rows)
	return rows
}

// ReceptaclePowersByReceptacleID 查询某插座的电量记录。
func ReceptaclePowersByReceptacleID(rid string) []model.ReceptaclePowerTbl {
	var rows []model.ReceptaclePowerTbl
	conf.Db.Where("receptacle_id = ?", rid).Find(&rows)
	return rows
}

// StaffByName 按姓名查询员工。
func StaffByName(name string) (*model.StaffTbl, error) {
	var staff model.StaffTbl
	if err := conf.Db.Where("name = ?", name).First(&staff).Error; err != nil {
		return nil, err
	}
	return &staff, nil
}

// ImeiTabByChargingStation 按充电站编号查询枪座绑定关系。
func ImeiTabByChargingStation(imei string) (*model.ImeiTab, error) {
	var row model.ImeiTab
	if err := conf.Db.Where("charging_station = ? AND deleted = 0", imei).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// CountUnreadMessages 统计某用户未读消息数。
func CountUnreadMessages(openid string) (int64, error) {
	var count int64
	err := conf.Db.Raw("SELECT count(*) FROM usermessage_tbl WHERE openid = ? AND status = '未读'", openid).Scan(&count).Error
	return count, err
}

// MessagesByOpenid 按时间倒序查询某用户消息。
func MessagesByOpenid(openid string) ([]model.UsermessageTbl, error) {
	var msgs []model.UsermessageTbl
	err := conf.Db.Where("openid = ?", openid).Order("msgTime desc").Find(&msgs).Error
	return msgs, err
}

// NearbyPlaces 查询附近可用车位（按经纬度距离排序，最多 100 条）。
func NearbyPlaces(longitude, latitude string) ([]model.PlaceTbl, error) {
	var places []model.PlaceTbl
	err := conf.Db.Raw(
		"SELECT * FROM place_tbl WHERE state = '可使用' AND longitude IS NOT NULL AND latitude IS NOT NULL "+
			"ORDER BY (POWER(MOD(ABS(longitude - ?), 360), 2) + POWER(ABS(latitude - ?), 2)) LIMIT 100",
		longitude, latitude,
	).Scan(&places).Error
	return places, err
}

// PlaceidsLike 按前缀查询 placeid 列表（倒序）。
func PlaceidsLike(prefix string) ([]string, error) {
	var placeids []string
	err := conf.Db.Raw("SELECT placeid FROM place_tbl WHERE placeid LIKE ? ORDER BY placeid DESC", prefix).Scan(&placeids).Error
	return placeids, err
}

// AllLocks 查询所有车位锁（lockid + lockmac）。
func AllLocks() ([]map[string]string, error) {
	var list []map[string]string
	err := conf.Db.Raw("SELECT lockid, lockmac FROM lock_tbl").Scan(&list).Error
	return list, err
}

// UpdateFaultState 更新故障状态，返回受影响行数。
func UpdateFaultState(faultid, state string) int64 {
	res := conf.Db.Model(&model.FaultTbl{}).Where("faultid = ?", faultid).Update("state", state)
	return res.RowsAffected
}

// UpdateFaultByLockmac 按锁 mac 更新故障记录，返回受影响行数。
func UpdateFaultByLockmac(lockmac string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.FaultTbl{}).Where("lockmac = ?", lockmac).Updates(updates)
	return res.RowsAffected
}

// InvoicesByOpenid 查询某用户发票。
func InvoicesByOpenid(openid string) []model.Invoice {
	var rows []model.Invoice
	conf.Db.Where("openid = ?", openid).Find(&rows)
	return rows
}

// InvoiceTitlesByOpenid 查询某用户发票抬头。
func InvoiceTitlesByOpenid(openid string) []model.InvoiceTitle {
	var rows []model.InvoiceTitle
	conf.Db.Where("openid = ?", openid).Find(&rows)
	return rows
}

// UpdateInvoiceStateByInvoiceID 按抬头 id 更新发票状态。
func UpdateInvoiceStateByInvoiceID(invoiceID string, state int) {
	conf.Db.Model(&model.Invoice{}).Where("invoice_id = ?", invoiceID).Update("state", state)
}
