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
