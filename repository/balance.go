package repository

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// ListUsersByBalance 分页查询有 openid/phone 的用户（按 openid 升序）。
func ListUsersByBalance(openid, phone string, offset, limit int) (int64, []model.UserTbl) {
	query := func() *gorm.DB {
		db := conf.Db.Model(&model.UserTbl{}).
			Where("openid IS NOT NULL AND openid <> '' AND phone IS NOT NULL AND phone <> ''")
		if openid != "" {
			db = db.Where("openid LIKE ?", "%"+openid+"%")
		}
		if phone != "" {
			db = db.Where("phone LIKE ?", "%"+phone+"%")
		}
		return db
	}
	var total int64
	query().Count(&total)
	var users []model.UserTbl
	query().Order("openid asc").Offset(offset).Limit(limit).Find(&users)
	return total, users
}

// PaidRechargeOrdersByOpenid 查询"已支付"且 remaining_refund_amount > 0 的充值订单。
func PaidRechargeOrdersByOpenid(openid string) []model.RechargeOrderTbl {
	var orders []model.RechargeOrderTbl
	conf.Db.Where("openid = ? AND order_status = ? AND remaining_refund_amount > ?", openid, "已支付", decimal.Zero).Find(&orders)
	return orders
}

// RechargeBalance 充值余额（事务内原子更新）。
func RechargeBalance(openid, amt string) error {
	return conf.Db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec(
			"UPDATE user_tbl SET balans = balans + CAST(? AS DECIMAL(10,2)) WHERE openid = ?",
			amt, openid,
		).Error
	})
}
