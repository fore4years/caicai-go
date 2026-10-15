package repository

import (
	"time"

	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/constant"
	"caicai-go/model"
)

// CreateRechargeOrder 创建充值订单。
func CreateRechargeOrder(order *model.RechargeOrderTbl) error {
	return conf.Db.Model(&model.RechargeOrderTbl{}).Omit("payment_time", "refund_time").Create(order).Error
}

// RechargeOrderByOrderID 按 order_id 查询充值订单。
func RechargeOrderByOrderID(orderID string) (*model.RechargeOrderTbl, error) {
	var rechargeOrder model.RechargeOrderTbl
	if err := conf.Db.Where("order_id = ?", orderID).First(&rechargeOrder).Error; err != nil {
		return nil, err
	}
	return &rechargeOrder, nil
}

// SettlePurchasePole 结算购桩订单，返回是否命中与错误。
func SettlePurchasePole(orderNumber, transactionID string) (bool, error) {
	var purchasePole model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", orderNumber).First(&purchasePole).Error; err != nil {
		return false, nil
	}
	res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
		Where("order_number = ?", orderNumber).
		Updates(map[string]interface{}{
			"transaction_id": transactionID,
			"pay_time":       time.Now(),
			"order_status":   "已支付",
		})
	if res.Error != nil {
		return true, res.Error
	}
	return true, nil
}

// SettlePackageRecord 结算套餐充值订单，返回是否命中与错误。
func SettlePackageRecord(orderNo, transactionID string) (bool, error) {
	var pkg model.PackageRecordTbl
	if err := conf.Db.Where("order_no = ?", orderNo).First(&pkg).Error; err != nil {
		return false, nil
	}
	updates := map[string]interface{}{
		"status":         "paid",
		"transaction_id": transactionID,
	}
	if pkg.EndTime.IsZero() && !pkg.CreateTime.IsZero() {
		updates["end_time"] = pkg.CreateTime.AddDate(0, 0, int(pkg.Days))
	}
	res := conf.Db.Model(&model.PackageRecordTbl{}).Where("order_no = ?", orderNo).Updates(updates)
	if res.Error != nil {
		return true, res.Error
	}
	return true, nil
}

// UpdateOrderWechatTransactionID 更新订单微信交易号，返回受影响行数与错误。
func UpdateOrderWechatTransactionID(orderid, transactionID string) (int64, error) {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderid).Update("wechat_transaction_id", transactionID)
	return res.RowsAffected, res.Error
}

// ProfitSharingConfigBySpacesID 按车位查询启用中的分账配置。
func ProfitSharingConfigBySpacesID(spacesID int32) (*model.TabProfitSharingConfig, error) {
	var config model.TabProfitSharingConfig
	if err := conf.Db.Where("spaces_id = ? AND enabled = ?", spacesID, true).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// SettleRechargeOrder 充值订单支付成功结算（事务）。
func SettleRechargeOrder(order *model.RechargeOrderTbl, transactionID string) error {
	if order.OrderStatus == "已支付" {
		return nil
	}
	return conf.Db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.RechargeOrderTbl{}).
			Where("order_id = ? AND order_status = ?", order.OrderID, constant.OrderRecharging).
			Updates(map[string]interface{}{
				"order_status":          "已支付",
				"wechat_transaction_id": transactionID,
				"payment_time":          time.Now(),
				"update_time":           time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected <= 0 {
			return nil
		}

		amt := order.RechargeAmount.StringFixed(2)
		if err := tx.Exec(
			"UPDATE user_tbl SET balans = balans + CAST(? AS DECIMAL(10,2)) WHERE openid = ?",
			amt, order.Openid,
		).Error; err != nil {
			return err
		}

		now := time.Now()
		return tx.Create(&model.TabBalanceRecord{
			Openid:     order.Openid,
			Type:       "充值",
			Amount:     order.RechargeAmount,
			CreateTime: &now,
		}).Error
	})
}
