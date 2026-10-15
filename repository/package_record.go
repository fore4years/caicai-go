package repository

import (
	"time"

	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// LatestPaidPackageRecord 查询最近一条已支付套餐记录。
func LatestPaidPackageRecord(openid string) model.PackageRecordTbl {
	var lastRecord model.PackageRecordTbl
	conf.Db.Where("openid = ? AND status = 'paid'", openid).
		Order("create_time desc").First(&lastRecord)
	return lastRecord
}

// DeductPackageBalance 仅扣可用余额 balans（单条原子 UPDATE，条件写进 WHERE 防止扣成负数）。
func DeductPackageBalance(openid string, amount decimal.Decimal) bool {
	amt := amount.StringFixed(2)
	res := conf.Db.Exec(
		"UPDATE user_tbl SET balans = balans - CAST(? AS DECIMAL(10,2)) WHERE openid = ? AND balans >= CAST(? AS DECIMAL(10,2))",
		amt, openid, amt,
	)
	return res.RowsAffected > 0
}

// PackageRecordByOrderNo 按 order_no 查询套餐记录。
func PackageRecordByOrderNo(orderNo string) (*model.PackageRecordTbl, error) {
	var pkg model.PackageRecordTbl
	if err := conf.Db.Where("order_no = ?", orderNo).First(&pkg).Error; err != nil {
		return nil, err
	}
	return &pkg, nil
}

// UpdatePackageRecordByOrderNo 按 order_no 更新套餐记录，返回受影响行数。
func UpdatePackageRecordByOrderNo(orderNo string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.PackageRecordTbl{}).Where("order_no = ?", orderNo).Updates(updates)
	return res.RowsAffected
}

// UpdatePackageRecordStatus 按 order_no 更新套餐状态。
func UpdatePackageRecordStatus(orderNo, status string) {
	conf.Db.Model(&model.PackageRecordTbl{}).Where("order_no = ?", orderNo).Update("status", status)
}

// LatestValidPackageByOpenid 查询最近一条未过期套餐。
func LatestValidPackageByOpenid(openid string) (*model.PackageRecordTbl, error) {
	var p model.PackageRecordTbl
	if err := conf.Db.Where("openid = ? AND status = 'paid' AND end_time > ?", openid, time.Now()).
		Order("create_time asc").First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ValidPackagesByOpenid 查询所有未过期套餐。
func ValidPackagesByOpenid(openid string) []model.PackageRecordTbl {
	var rows []model.PackageRecordTbl
	conf.Db.Where("openid = ? AND status = 'paid' AND end_time > ?", openid, time.Now()).
		Order("create_time asc").Find(&rows)
	return rows
}

// MaxPowerFromRecentOrders 查询最近一周二轮车订单的最大功率。
func MaxPowerFromRecentOrders(openid string) decimal.Decimal {
	var maxPower decimal.Decimal
	conf.Db.Raw(`SELECT COALESCE(MAX(re_order_electronic_tbl.max_power), 0)
		FROM order_tbl
		LEFT JOIN re_order_electronic_tbl ON order_tbl.orderid = re_order_electronic_tbl.order_id
		WHERE order_tbl.openid = ?
		  AND order_tbl.consumption_type = '二轮车充电'
		  AND order_tbl.close_time >= DATE_SUB(NOW(), INTERVAL 7 DAY)`, openid).Scan(&maxPower)
	return maxPower
}
