package repository

import (
	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// 定时任务所需的查询/更新，对齐 Java ExamineOrder / RefundStatusSyncTask / RenewalBooking。
// ============================================================================

// AllChargingByBooking 查询所有驿享充电的预约订单（"预约中"或"已预约"），
// 对齐 Java OrderServiceImpl.getAllChargingByBooking。
func AllChargingByBooking() []model.OrderTbl {
	var orders []model.OrderTbl
	conf.Db.Where("consumption_type = ? AND (state = ? OR state = ?)",
		ConsumeTypeYiXiang, OrderStateBooking, OrderStateBooked).Find(&orders)
	return orders
}

// OrdersByStateAndConsumption 按消费类型 + 状态查询订单，对齐 Java lambdaQuery 组合条件。
func OrdersByStateAndConsumption(consumption, state string) []model.OrderTbl {
	var orders []model.OrderTbl
	conf.Db.Where("consumption_type = ? AND state = ?", consumption, state).Find(&orders)
	return orders
}

// OrdersByState 按状态查询订单。
func OrdersByState(state string) []model.OrderTbl {
	var orders []model.OrderTbl
	conf.Db.Where("state = ?", state).Find(&orders)
	return orders
}

// RenewalBookingAll 每天 0 点重置所有用户预约次数（对齐 Java DriverUseDao.updateRenewalBooking）。
// 注意：Java 原始 SQL 无 WHERE 条件，会更新全表。
func RenewalBookingAll(reserveNumber int32, everyDay string) int64 {
	res := conf.Db.Model(&model.DriveruseTbl{}).Where("1 = 1").Updates(map[string]interface{}{
		"reserveNumber": reserveNumber,
		"everyDay":      everyDay,
	})
	return res.RowsAffected
}

// PendingRefundRecords 查询待同步状态的退款记录。
// statuses 为需要同步的 withdrawal_status 集合；excludeSuccess 为 true 时排除已成功记录（全量同步用）。
func PendingRefundRecords(statuses []string, excludeSuccess bool) []model.WithdrawalRecordTbl {
	q := conf.Db.Model(&model.WithdrawalRecordTbl{}).
		Where("withdrawal_type = ?", "退款").
		Where("refund_order_id IS NOT NULL AND refund_order_id <> ''")
	if len(statuses) > 0 {
		q = q.Where("withdrawal_status IN ?", statuses)
	}
	if excludeSuccess {
		q = q.Where("withdrawal_status <> ?", "已处理")
	}
	var records []model.WithdrawalRecordTbl
	q.Find(&records)
	return records
}
