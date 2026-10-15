package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// SearchGateways 按字段模糊搜索网关。
func SearchGateways(field, keyword string, offset, limit int) (int64, []model.TabGateway) {
	var max int64
	conf.Db.Model(&model.TabGateway{}).Where(field+" LIKE ?", "%"+keyword+"%").Count(&max)
	var gateways []model.TabGateway
	conf.Db.Model(&model.TabGateway{}).Where(field+" LIKE ?", "%"+keyword+"%").Order("install_time desc").Offset(offset).Limit(limit).Find(&gateways)
	return max, gateways
}

// PageGatewaysByMac 按 mac 模糊分页查询网关。
func PageGatewaysByMac(mac string, offset, limit int) (int64, []model.TabGateway) {
	var max int64
	conf.Db.Model(&model.TabGateway{}).Where("mac LIKE ?", "%"+mac+"%").Count(&max)
	var gateways []model.TabGateway
	conf.Db.Model(&model.TabGateway{}).Where("mac LIKE ?", "%"+mac+"%").Order("install_time desc").Offset(offset).Limit(limit).Find(&gateways)
	return max, gateways
}

// SearchWangguanByAzadd 按安装地址模糊搜索 wangguan_tbl。
func SearchWangguanByAzadd(azadd string, offset, limit int) (int64, []model.WangguanTbl) {
	var max int64
	conf.Db.Model(&model.WangguanTbl{}).Where("azadd LIKE ?", "%"+azadd+"%").Count(&max)
	var rows []model.WangguanTbl
	conf.Db.Model(&model.WangguanTbl{}).Where("azadd LIKE ?", "%"+azadd+"%").Offset(offset).Limit(limit).Find(&rows)
	return max, rows
}

// AllCharges 分页查询 charge_tbl。
func AllCharges(offset, limit int) []model.ChargeTbl {
	var rows []model.ChargeTbl
	conf.Db.Raw("SELECT placeid, charge FROM charge_tbl LIMIT ?,?", offset, limit).Scan(&rows)
	return rows
}

// LocksByState 按车位状态查询车位锁（左连 place_tbl）。
func LocksByState(state string) []model.LockTbl {
	db := conf.Db.Table("lock_tbl l").Select("l.*").Joins("left join place_tbl p on l.placeid = p.placeid")
	if state == "已禁用" {
		db = db.Where("p.state = ?", "已禁用")
	} else {
		db = db.Where("p.state <> ?", "已禁用")
	}
	var locks []model.LockTbl
	db.Scan(&locks)
	return locks
}

// SearchLocks 按字段模糊搜索车位锁。
func SearchLocks(field, keyword string, offset, limit int) (int64, []model.LockTbl) {
	var max int64
	conf.Db.Model(&model.LockTbl{}).Where(field+" LIKE ?", "%"+keyword+"%").Count(&max)
	var locks []model.LockTbl
	conf.Db.Model(&model.LockTbl{}).Where(field+" LIKE ?", "%"+keyword+"%").Offset(offset).Limit(limit).Find(&locks)
	return max, locks
}

// UpdatePlaceIscharge 标记车位已装充电桩。
func UpdatePlaceIscharge(placeid string) {
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Update("ischarge", 1)
}

// UpdateChargeStateByPlaceid 更新充电桩状态，返回受影响行数。
func UpdateChargeStateByPlaceid(placeid, state string) int64 {
	res := conf.Db.Exec("UPDATE charge_tbl SET charge = ? WHERE placeid = ?", state, placeid)
	return res.RowsAffected
}

// FaultByFaultid 按 faultid 查询故障。
func FaultByFaultid(faultid string) (*model.FaultTbl, error) {
	var f model.FaultTbl
	if err := conf.Db.Where("faultid = ?", faultid).First(&f).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

// DeleteFaultByFaultid 删除故障。
func DeleteFaultByFaultid(faultid string) {
	conf.Db.Where("faultid = ?", faultid).Delete(&model.FaultTbl{})
}

// SearchFaults 按字段模糊搜索故障。
func SearchFaults(field, keyword string, offset, limit int) (int64, []model.FaultTbl) {
	var max int64
	conf.Db.Model(&model.FaultTbl{}).Where(field+" LIKE ?", "%"+keyword+"%").Count(&max)
	var faults []model.FaultTbl
	conf.Db.Model(&model.FaultTbl{}).Where(field+" LIKE ?", "%"+keyword+"%").Offset(offset).Limit(limit).Find(&faults)
	return max, faults
}
