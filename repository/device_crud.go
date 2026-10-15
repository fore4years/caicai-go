package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// LockByProductID 按 product_id 查询车位锁。
func LockByProductID(productID string) (*model.LockTbl, error) {
	var l model.LockTbl
	if err := conf.Db.Where("product_id = ?", productID).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// UpdateChargingGunPwm 更新充电枪 pwm，返回受影响行数。
func UpdateChargingGunPwm(productID, direction string, pwm int) int64 {
	res := conf.Db.Model(&model.TabChargingGun{}).
		Where("product_id = ? AND direction = ?", productID, direction).
		Update("pwm", pwm)
	return res.RowsAffected
}

// StationManagementsByOpenid 按 openid 查询充电站管理列表。
func StationManagementsByOpenid(openid string) []model.ChargingStationManagementTbl {
	var rows []model.ChargingStationManagementTbl
	conf.Db.Where("openid = ?", openid).Find(&rows)
	return rows
}

// StationManagementExistsByProductID 判断充电站管理是否存在。
func StationManagementExistsByProductID(productID string) bool {
	var cnt int64
	conf.Db.Model(&model.ChargingStationManagementTbl{}).Where("product_id = ?", productID).Count(&cnt)
	return cnt > 0
}

// ReceptaclesByPid 按 pid 查询插座列表。
func ReceptaclesByPid(pid string) []model.ReceptacleTbl {
	var rows []model.ReceptacleTbl
	conf.Db.Where("pid = ?", pid).Find(&rows)
	return rows
}

// DeleteReceptaclesByPid 按 pid 删除插座。
func DeleteReceptaclesByPid(pid string) {
	conf.Db.Where("pid = ?", pid).Delete(&model.ReceptacleTbl{})
}

// ImeiBindingExists 判断 imei 绑定是否存在。
func ImeiBindingExists(station, socket string) bool {
	var cnt int64
	conf.Db.Model(&model.ImeiTab{}).
		Where("charging_station = ? AND charging_socket = ? AND deleted = 0", station, socket).
		Count(&cnt)
	return cnt > 0
}

// DeleteImeiBinding 逻辑删除 imei 绑定，返回受影响行数。
func DeleteImeiBinding(station, socket string) int64 {
	res := conf.Db.Model(&model.ImeiTab{}).
		Where("charging_station = ? AND charging_socket = ? AND deleted = 0", station, socket).
		Update("deleted", 1)
	return res.RowsAffected
}

// ListImeiBindings 分页查询未删除的 imei 绑定。
func ListImeiBindings(offset, limit int) (int64, []model.ImeiTab) {
	var total int64
	conf.Db.Model(&model.ImeiTab{}).Where("deleted = 0").Count(&total)
	var rows []model.ImeiTab
	conf.Db.Model(&model.ImeiTab{}).Where("deleted = 0").Offset(offset).Limit(limit).Find(&rows)
	return total, rows
}
