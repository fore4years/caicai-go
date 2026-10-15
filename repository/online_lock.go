package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// PlaceLockLive 查询车位锁 live 状态。
func PlaceLockLive(lockID string) (string, bool) {
	var live string
	if err := conf.Db.Raw("SELECT live FROM place_Lock_Table WHERE lock_ID = ?", lockID).Row().Scan(&live); err != nil {
		return "", false
	}
	return live, true
}

// OnlineLockByLockID 按 lockID 查询锁在线状态。
func OnlineLockByLockID(lockID string) (*model.OnlineLockstatusTable, error) {
	var on model.OnlineLockstatusTable
	if err := conf.Db.Where("lockID = ?", lockID).First(&on).Error; err != nil {
		return nil, err
	}
	return &on, nil
}

// HasUnfinishedOrders 判断用户是否有未完成订单。
func HasUnfinishedOrders(openid string) bool {
	var unfinished []model.OrderTbl
	conf.Db.Where("openid = ? AND state <> ? AND state <> ?", openid, OrderStateFinished, OrderStateRevoked).Find(&unfinished)
	return len(unfinished) > 0
}

// UpdateOnlineLockOnline 更新锁在线状态，返回受影响行数。
func UpdateOnlineLockOnline(lockID, online string) int64 {
	res := conf.Db.Model(&model.OnlineLockstatusTable{}).Where("lockID = ?", lockID).Update("online", online)
	return res.RowsAffected
}

// UpdateOnlineLockBattery 更新锁电量状态，返回受影响行数。
func UpdateOnlineLockBattery(lockID, battery string) int64 {
	res := conf.Db.Model(&model.OnlineLockstatusTable{}).Where("lockID = ?", lockID).Update("battery", battery)
	return res.RowsAffected
}

// MaxOnlineLockNumber 查询锁最大编号。
func MaxOnlineLockNumber() string {
	var maxNumStr string
	conf.Db.Raw("SELECT number FROM online_lockstatus_Table ORDER BY number DESC LIMIT 0,1").Row().Scan(&maxNumStr)
	return maxNumStr
}

// InsertOnlineLock 插入锁上线记录，返回受影响行数。
func InsertOnlineLock(number, lockID string) int64 {
	res := conf.Db.Exec("INSERT INTO online_lockstatus_Table (number, lockID, battery, online) VALUES (?, ?, 'y', 'n')", number, lockID)
	return res.RowsAffected
}
