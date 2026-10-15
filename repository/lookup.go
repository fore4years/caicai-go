package repository

import (
	"time"

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

// FindOrderByOrderid 按 orderid 查询订单。
func FindOrderByOrderid(orderid string) (*model.OrderTbl, error) {
	var order model.OrderTbl
	if err := conf.Db.Where("orderid = ?", orderid).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// CountPrivateChargingByPhone 统计某手机号私桩数量。
func CountPrivateChargingByPhone(phone string) int64 {
	var total int64
	conf.Db.Model(&model.PrivateChargingTbl{}).Where("phone LIKE ?", "%"+phone+"%").Count(&total)
	return total
}

// PagePrivateChargingByPhone 分页查询某手机号私桩。
func PagePrivateChargingByPhone(phone string, offset, limit int) []model.PrivateChargingTbl {
	var rows []model.PrivateChargingTbl
	conf.Db.Model(&model.PrivateChargingTbl{}).Where("phone LIKE ?", "%"+phone+"%").
		Offset(offset).Limit(limit).Find(&rows)
	return rows
}

// UpdatePrivateChargingStatus 更新私桩审核状态，返回受影响行数。
func UpdatePrivateChargingStatus(id, status int32) int64 {
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id = ?", id).Update("status", status)
	return res.RowsAffected
}

// UpdatePrivateChargingStatusByIDs 批量更新私桩审核状态，返回受影响行数。
func UpdatePrivateChargingStatusByIDs(ids []int32, status int32) int64 {
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id IN ?", ids).Update("status", status)
	return res.RowsAffected
}

// OrdersByOpenid 按开始时间倒序分页查询某用户订单。
func OrdersByOpenid(openid string, offset, limit int) ([]model.OrderTbl, error) {
	var orders []model.OrderTbl
	err := conf.Db.Where("openid = ?", openid).Order("begin_time desc").Limit(limit).Offset(offset).Find(&orders).Error
	return orders, err
}

// PlaceByOrderidJoin 按订单关联查询车位（order→lock→place），返回是否找到。
func PlaceByOrderidJoin(orderid string) (*model.PlaceTbl, bool) {
	var place model.PlaceTbl
	res := conf.Db.Raw(
		"SELECT p.* FROM order_tbl o JOIN lock_tbl l ON o.lockid = l.lockid JOIN place_tbl p ON p.placeid = l.placeid WHERE o.orderid = ?",
		orderid,
	).Scan(&place)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, false
	}
	return &place, true
}

// UserPlateAndPhone 查询用户车牌号与手机号。
func UserPlateAndPhone(openid string) (string, string, error) {
	var msg struct {
		PlateNum string
		Phone    string
	}
	err := conf.Db.Raw("SELECT plate_num, phone FROM user_tbl WHERE openid = ?", openid).Scan(&msg).Error
	return msg.PlateNum, msg.Phone, err
}

// UpdateOrderFields 按 orderid 更新订单字段，返回受影响行数。
func UpdateOrderFields(orderid string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderid).Updates(updates)
	return res.RowsAffected
}

// FindChargeOrderByOrderid 按 orderid 查询充电订单。
func FindChargeOrderByOrderid(orderid string) (*model.ChargeOrderTbl, error) {
	var o model.ChargeOrderTbl
	if err := conf.Db.Where("orderid = ?", orderid).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// UpdateChargeOrderState 更新充电订单状态，返回受影响行数。
func UpdateChargeOrderState(orderid, state string) int64 {
	res := conf.Db.Model(&model.ChargeOrderTbl{}).Where("orderid = ?", orderid).Update("state", state)
	return res.RowsAffected
}

// InvoicesStatePaged 分页查询 state IN (1,2) 的发票（按发票日期降序）。
func InvoicesStatePaged(openid string, offset, limit int) (int64, []model.Invoice) {
	base := conf.Db.Model(&model.Invoice{}).Where("state IN ?", []int32{1, 2})
	if openid != "" {
		base = base.Where("openid = ?", openid)
	}
	var total int64
	base.Count(&total)
	var invoices []model.Invoice
	base.Order("invoice_date desc").Offset(offset).Limit(limit).Find(&invoices)
	return total, invoices
}

// UsersByPhoneAll 查询某手机号的所有用户。
func UsersByPhoneAll(phone string) []model.UserTbl {
	var users []model.UserTbl
	conf.Db.Where("phone = ?", phone).Find(&users)
	return users
}

// UsersByOpenids 批量查询用户。
func UsersByOpenids(openids []string) []model.UserTbl {
	var users []model.UserTbl
	conf.Db.Where("openid IN ?", openids).Find(&users)
	return users
}

// InvoiceTitlesByOpenids 批量查询发票抬头。
func InvoiceTitlesByOpenids(openids []string) []model.InvoiceTitle {
	var titles []model.InvoiceTitle
	conf.Db.Where("openid IN ?", openids).Find(&titles)
	return titles
}

// UpdateInvoiceStateByID 按 id 更新发票状态，返回受影响行数。
func UpdateInvoiceStateByID(id string, state int) int64 {
	res := conf.Db.Model(&model.Invoice{}).Where("id = ?", id).Update("state", state)
	return res.RowsAffected
}

// DeleteInvoicesByIDs 批量删除发票，返回受影响行数。
func DeleteInvoicesByIDs(ids []int32) int64 {
	res := conf.Db.Where("id IN ?", ids).Delete(&model.Invoice{})
	return res.RowsAffected
}

// HasRecentInvoiceApply 判断最近两天是否有待处理发票申请。
func HasRecentInvoiceApply() bool {
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	var n int64
	conf.Db.Model(&model.Invoice{}).
		Where("state = 1 AND apply_time >= ? AND apply_time <= ?", twoDaysAgo, time.Now()).
		Count(&n)
	return n > 0
}

// FindUserByOpenid 按 openid 查询用户（未找到返回 nil）。
func FindUserByOpenid(openid string) *model.UserTbl {
	var u model.UserTbl
	if conf.Db.Where("openid = ?", openid).First(&u).Error != nil {
		return nil
	}
	return &u
}

// UserBankCardByOpenid 按 openid 查询银行卡。
func UserBankCardByOpenid(openid string) (*model.UserBankCardTbl, error) {
	var m model.UserBankCardTbl
	if err := conf.Db.Where("openid = ?", openid).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// UserBluetoothBindingsByOpenid 按 openid 查询蓝牙绑定。
func UserBluetoothBindingsByOpenid(openid string) (*model.UserBluetoothBindingsTbl, error) {
	var m model.UserBluetoothBindingsTbl
	if err := conf.Db.Where("openid = ?", openid).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// AccountPublicByOrderID 按 order_id 查询公司账户。
func AccountPublicByOrderID(orderID string) (*model.AccountPublicTbl, error) {
	var a model.AccountPublicTbl
	if err := conf.Db.Where("order_id = ?", orderID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// AccountPrivateByOrderID 按 order_id 查询私人账户。
func AccountPrivateByOrderID(orderID string) (*model.AccountPrivateTbl, error) {
	var a model.AccountPrivateTbl
	if err := conf.Db.Where("order_id = ?", orderID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// BalanceRecordsByOpenid 按 openid 分页查询余额明细。
func BalanceRecordsByOpenid(openid string, offset, limit int) (int64, []model.TabBalanceRecord) {
	var total int64
	conf.Db.Model(&model.TabBalanceRecord{}).Where("openid = ?", openid).Count(&total)
	var rows []model.TabBalanceRecord
	conf.Db.Model(&model.TabBalanceRecord{}).Where("openid = ?", openid).Order("create_time desc").Offset(offset).Limit(limit).Find(&rows)
	return total, rows
}
