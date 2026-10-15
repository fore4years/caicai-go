package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// 本文件对应 handler/order_crud.go（OrderElectronic / OrderPrivate / OrderTwice /
// RechargeOrder / Withdrawal 控制器）的内联 SQL，统一抽离为命名函数。
// ============================================================================

// TabOrderElectronicPeriodRow 订单各时段电费/服务费明细扫描结构。
type TabOrderElectronicPeriodRow struct {
	CreateTime       time.Time
	UpdateTime       time.Time
	PriceTotal       sql.NullString
	TotalFee         sql.NullString
	ServiceUnitPrice sql.NullString
	ServiceFee       sql.NullString
}

// ---------------------------- OrderElectronic ----------------------------

// TabOrderElectronicByID 按 id 查询订单充电时段电量。
func TabOrderElectronicByID(id string) (*model.TabOrderElectronic, error) {
	var e model.TabOrderElectronic
	if err := conf.Db.Where("id = ?", id).First(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// InsertTabOrderElectronic 新增订单充电时段电量。
func InsertTabOrderElectronic(e *model.TabOrderElectronic) error {
	return conf.Db.Create(e).Error
}

// PageTabOrderElectronics 分页查询订单充电时段电量。
func PageTabOrderElectronics(offset, limit int) (int64, []model.TabOrderElectronic) {
	var total int64
	conf.Db.Model(&model.TabOrderElectronic{}).Count(&total)
	var rows []model.TabOrderElectronic
	conf.Db.Model(&model.TabOrderElectronic{}).Offset(offset).Limit(limit).Find(&rows)
	return total, rows
}

// TabOrderElectronicValueByOrderID 查询订单电量差（MAX(over_value)-MIN(start_value)）。
func TabOrderElectronicValueByOrderID(orderID string) *decimal.Decimal {
	var d decimal.Decimal
	if err := conf.Db.Raw("SELECT (MAX(over_value)-MIN(start_value)) FROM tab_order_electronic WHERE order_id = ?", orderID).Row().Scan(&d); err != nil {
		return nil
	}
	return &d
}

// TabOrderElectronicFeesByOrderID 查询订单电费 + 服务费合计（tab_price + tab_parking_spaces）。
func TabOrderElectronicFeesByOrderID(orderID string) *decimal.Decimal {
	season := CurrentSeason()
	var d decimal.Decimal
	if err := conf.Db.Raw(`SELECT SUM((e.over_value - e.start_value) / 100 * (
			CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END
			+
			CASE
				WHEN s.service_fee IS NOT NULL AND s.service_fee != 0 THEN s.service_fee
				WHEN s.service IS NOT NULL AND s.service != 0 THEN (s.service * 0.8)
				ELSE (p.service * 0.8)
			END
		))
		FROM tab_order_electronic e
		LEFT JOIN tab_price_time_`+season+` t ON t.id = e.price_time_id
		LEFT JOIN tab_price p ON p.id = t.price_id
		LEFT JOIN order_tbl o ON o.orderid = e.order_id
		LEFT JOIN tab_parking_spaces s ON s.id = o.spaces_id
		WHERE e.order_id = ?`, orderID).Row().Scan(&d); err != nil {
		return nil
	}
	return &d
}

// ElectronicFeesByOrderid 计算订单的电费与服务费（对齐 Java getElectronicByOrderid，季节电价表）。
func ElectronicFeesByOrderid(orderID string) (totalFee, serviceFee decimal.Decimal) {
	season := CurrentSeason()
	sql := `SELECT
		COALESCE(SUM((e.over_value - e.start_value) / 100 * CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END),0),
		COALESCE(SUM((e.over_value - e.start_value) / 100 * CASE WHEN s.service_fee IS NOT NULL AND s.service_fee != 0 THEN s.service_fee WHEN s.service IS NOT NULL AND s.service != 0 THEN (s.service*0.8) ELSE (p.service*0.8) END),0)
		FROM tab_order_electronic e
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = e.price_time_id
		LEFT JOIN tab_price p ON p.id = t.price_id
		LEFT JOIN order_tbl o ON o.orderid = e.order_id
		LEFT JOIN tab_parking_spaces s ON s.id = o.spaces_id
		WHERE e.order_id = ?`
	var tf, sf decimal.Decimal
	conf.Db.Raw(sql, orderID).Row().Scan(&tf, &sf)
	return tf.Round(2), sf.Round(2)
}

// TabOrderElectronicPeriodsByOrderID 查询订单各时段电费/服务费明细。
func TabOrderElectronicPeriodsByOrderID(orderID string) ([]TabOrderElectronicPeriodRow, error) {
	season := CurrentSeason()
	rows, err := conf.Db.Raw(`SELECT
			e.create_time,
			e.update_time,
			CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END AS price_total,
			(e.over_value - e.start_value) / 100 * CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END AS total_fee,
			CASE
				WHEN s.service_fee IS NOT NULL AND s.service_fee != 0 THEN s.service_fee
				WHEN s.service IS NOT NULL AND s.service != 0 THEN (s.service * 0.8)
				ELSE (p.service * 0.8)
			END AS service_unit_price,
			(e.over_value - e.start_value) / 100 * CASE
				WHEN s.service_fee IS NOT NULL AND s.service_fee != 0 THEN s.service_fee
				WHEN s.service IS NOT NULL AND s.service != 0 THEN (s.service * 0.8)
				ELSE (p.service * 0.8)
			END AS service_fee
		FROM tab_order_electronic e
		LEFT JOIN tab_price_time_`+season+` t ON t.id = e.price_time_id
		LEFT JOIN tab_price p ON p.id = t.price_id
		LEFT JOIN order_tbl o ON o.orderid = e.order_id
		LEFT JOIN tab_parking_spaces s ON s.id = o.spaces_id
		WHERE e.order_id = ? AND e.over_value != e.start_value
		ORDER BY e.create_time`, orderID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]TabOrderElectronicPeriodRow, 0)
	for rows.Next() {
		var r TabOrderElectronicPeriodRow
		if err := rows.Scan(&r.CreateTime, &r.UpdateTime, &r.PriceTotal, &r.TotalFee, &r.ServiceUnitPrice, &r.ServiceFee); err != nil {
			continue
		}
		list = append(list, r)
	}
	return list, nil
}

// ---------------------------- OrderPrivate ----------------------------

// InsertOrderPrivate 新增私桩订单计费记录。
func InsertOrderPrivate(o *model.OrderPrivateTbl) error {
	return conf.Db.Create(o).Error
}

// ---------------------------- OrderTwice ----------------------------

// OrderTwiceByOrderid 按订单 id 查询二轮车订单计费记录。
func OrderTwiceByOrderid(orderid string) (*model.OrderTwiceTbl, error) {
	var o model.OrderTwiceTbl
	if err := conf.Db.Where("orderid = ?", orderid).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// InsertOrderTwice 新增二轮车订单计费记录。
func InsertOrderTwice(o *model.OrderTwiceTbl) error {
	return conf.Db.Create(o).Error
}

// ---------------------------- RechargeOrder ----------------------------

// RechargeOrdersByOpenidStatus 按 openid + 状态查询充值订单（创建时间倒序）。
func RechargeOrdersByOpenidStatus(openid, status string) []model.RechargeOrderTbl {
	var rows []model.RechargeOrderTbl
	conf.Db.Where("openid = ? AND order_status = ?", openid, status).Order("create_time desc").Find(&rows)
	return rows
}

// RechargeOrderSumsByOpenid 汇总用户充值金额与已退款金额。
func RechargeOrderSumsByOpenid(openid string) (decimal.Decimal, decimal.Decimal) {
	var totalAmount, refundedAmount decimal.Decimal
	conf.Db.Model(&model.RechargeOrderTbl{}).Where("openid = ?", openid).
		Select("COALESCE(SUM(recharge_amount),0), COALESCE(SUM(refunded_amount),0)").Row().Scan(&totalAmount, &refundedAmount)
	return totalAmount, refundedAmount
}

// CountRechargeOrdersByOpenid 统计用户充值订单数量。
func CountRechargeOrdersByOpenid(openid string) int64 {
	var count int64
	conf.Db.Model(&model.RechargeOrderTbl{}).Where("openid = ?", openid).Count(&count)
	return count
}

// UpdateRechargeOrderStatus 更新充值订单状态，返回受影响行数。
func UpdateRechargeOrderStatus(orderID, status string) int64 {
	res := conf.Db.Model(&model.RechargeOrderTbl{}).Where("order_id = ?", orderID).Update("order_status", status)
	return res.RowsAffected
}

// UpdateRefundApplyCount 更新充值订单退款申请次数，返回受影响行数。
func UpdateRefundApplyCount(orderID string, count int32) int64 {
	res := conf.Db.Model(&model.RechargeOrderTbl{}).Where("order_id = ?", orderID).Update("refund_apply_count", count)
	return res.RowsAffected
}

// ---------------------------- Withdrawal ----------------------------

// InsertWithdrawalRecord 新增提现/退款记录。
func InsertWithdrawalRecord(rec *model.WithdrawalRecordTbl) error {
	return conf.Db.Create(rec).Error
}

// WithdrawalRecordByRecordID 按记录 id 查询提现/退款记录。
func WithdrawalRecordByRecordID(recordID string) (*model.WithdrawalRecordTbl, error) {
	var w model.WithdrawalRecordTbl
	if err := conf.Db.Where("record_id = ?", recordID).First(&w).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

// WithdrawalRecordsByOpenid 按 openid 查询提现/退款记录（创建时间倒序）。
func WithdrawalRecordsByOpenid(openid string) []model.WithdrawalRecordTbl {
	var rows []model.WithdrawalRecordTbl
	conf.Db.Where("openid = ?", openid).Order("create_time desc").Find(&rows)
	return rows
}

// LatestWithdrawalRecord 查询刚创建的退款记录（openid + 关联充值订单 + 金额，创建时间倒序取首条）。
func LatestWithdrawalRecord(openid, relatedRechargeOrderID string, amount decimal.Decimal) (*model.WithdrawalRecordTbl, error) {
	var latest model.WithdrawalRecordTbl
	if err := conf.Db.Where("openid = ? AND related_recharge_order_id = ? AND withdrawal_amount = ?",
		openid, relatedRechargeOrderID, amount).
		Order("create_time desc").First(&latest).Error; err != nil {
		return nil, err
	}
	return &latest, nil
}

// UpdateWithdrawalRecord 按记录 id 更新提现/退款记录，返回受影响行数。
func UpdateWithdrawalRecord(recordID string, updates map[string]interface{}) int64 {
	res := conf.Db.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", recordID).Updates(updates)
	return res.RowsAffected
}

// ---------------------------- 私桩费率 ----------------------------

// PrivateChargingFeeByOrderid 根据订单查询桩主设置的服务费率（order→private_place→private_charging）。
func PrivateChargingFeeByOrderid(orderID string) *decimal.Decimal {
	order, err := FindOrderByOrderid(orderID)
	if err != nil {
		return nil
	}
	privatePlace := PrivatePlaceByID(order.SpacesID)
	if privatePlace == nil {
		return nil
	}
	chargingBean := PrivateChargingBySpacesNum(privatePlace.SpacesCode)
	if chargingBean == nil {
		return nil
	}
	fee := chargingBean.Fee
	return &fee
}

// ---------------------------- 余额退款事务 ----------------------------

// BalanceRefundTx 余额退款事务：扣减余额 + 更新退款记录 + 更新充值订单退款信息 + 记录余额明细。
func BalanceRefundTx(refundRecord *model.WithdrawalRecordTbl, rechargeOrder *model.RechargeOrderTbl) error {
	refundAmount := refundRecord.WithdrawalAmount
	amt := refundAmount.StringFixed(2)

	return conf.Db.Transaction(func(tx *gorm.DB) error {
		// 1. 扣减用户余额（原子 UPDATE，条件写进 WHERE 防止扣成负数）
		res := tx.Exec(
			"UPDATE user_tbl SET balans = balans - CAST(? AS DECIMAL(10,2)) WHERE openid = ? AND balans >= CAST(? AS DECIMAL(10,2))",
			amt, refundRecord.Openid, amt,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected <= 0 {
			return fmt.Errorf("退款金额超过用户当前余额")
		}

		// 2. 更新退款记录状态
		if err := tx.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", refundRecord.RecordID).
			Updates(map[string]interface{}{"withdrawal_status": "已处理", "process_time": time.Now()}).Error; err != nil {
			return err
		}

		// 3. 更新充值订单的退款信息
		rechargeOrder.RefundedAmount = rechargeOrder.RefundedAmount.Add(refundAmount)
		rechargeOrder.RemainingRefundAmount = rechargeOrder.RemainingRefundAmount.Sub(refundAmount)
		if err := tx.Model(&model.RechargeOrderTbl{}).Where("order_id = ?", rechargeOrder.OrderID).
			Updates(map[string]interface{}{
				"refunded_amount":         rechargeOrder.RefundedAmount,
				"remaining_refund_amount": rechargeOrder.RemainingRefundAmount,
			}).Error; err != nil {
			return err
		}

		// 4. 记录余额变动明细
		if err := tx.Create(&model.TabBalanceRecord{Openid: refundRecord.Openid, Type: "退费", Amount: refundAmount}).Error; err != nil {
			return err
		}

		return nil
	})
}
