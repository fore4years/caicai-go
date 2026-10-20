package repository

import (
	"strings"
	"time"

	"github.com/google/uuid"
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

// RechargeBalance 充值余额（事务内：加余额 + 建充值订单 + 记余额明细）。
func RechargeBalance(openid, amt string) error {
	amount, err := decimal.NewFromString(amt)
	if err != nil {
		return err
	}
	now := time.Now()
	orderID := strings.ReplaceAll(uuid.NewString(), "-", "")
	return conf.Db.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(
			"UPDATE user_tbl SET balans = balans + CAST(? AS DECIMAL(10,2)) WHERE openid = ?",
			amt, openid,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected <= 0 {
			return gorm.ErrRecordNotFound
		}

		// 建充值订单（已支付、可退），供「充值记录/退费」界面展示
		if err := tx.Create(&model.RechargeOrderTbl{
			OrderID:               orderID,
			Openid:                openid,
			RechargeAmount:        amount,
			RefundedAmount:        decimal.Zero,
			RemainingRefundAmount: amount,
			MaxRefundAmount:       amount,
			OrderStatus:           "已支付",
			CreateTime:            &now,
			UpdateTime:            &now,
			PaymentTime:           &now,
		}).Error; err != nil {
			return err
		}

		// 记余额明细
		return tx.Create(&model.TabBalanceRecord{
			Openid:     openid,
			Type:       "充值",
			Amount:     amount,
			CreateTime: &now,
		}).Error
	})
}
