package service

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/repository"
)

// ============================================================================
// 定时任务（对齐 Java util/ExamineOrder.java、RefundStatusSyncTask.java、RenewalBooking.java）。
// 用 time.Ticker / 每日定时器替代 Spring @Scheduled。
// ============================================================================

// 退款状态字面量（对齐 Java RefundStatusEnum）。
const (
	refundStatusApplying   = "退款申请中"
	refundStatusProcessing = "处理中"
	refundStatusSuccess    = "已处理"
	refundStatusRejected   = "已拒绝"
	refundStatusClosed     = "已关闭"
)

// StartScheduledTasks 启动所有定时任务，ctx 取消时全部退出（用于优雅关闭）。
func StartScheduledTasks(ctx context.Context) {
	// ExamineOrder
	go runFixedDelay(ctx, 10*time.Minute, cancelChargingOrder)
	go runFixedDelay(ctx, 5*time.Minute, cancelChargingOrderIn15Minutes)
	go runFixedDelay(ctx, 5*time.Minute, cancelTwoWheelChargingOrderIn15Minutes)
	go runFixedDelay(ctx, 5*time.Minute, closeBalanceNotEnoughOrder)
	// RefundStatusSyncTask
	go runFixedDelay(ctx, 30*time.Minute, syncRefundStatus)
	go runDaily(ctx, 2, 0, fullSyncRefundStatus)
	// RenewalBooking
	go runDaily(ctx, 0, 0, renewalBooking)
	logger.Mylog.Info().Msg("定时任务已启动（订单超时取消/余额不足断电/退款同步/预约续期）")
}

// safeRun 包装任务执行，捕获 panic，避免单次任务异常拖垮调度循环。
func safeRun(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logger.Mylog.Error().Msgf("[scheduler] 任务 panic: %v", r)
		}
	}()
	fn()
}

// runFixedDelay 以固定延迟调度（对齐 @Scheduled(fixedDelay)）：先执行一次，之后每次间隔 interval。
func runFixedDelay(ctx context.Context, interval time.Duration, fn func()) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		safeRun(fn)
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// runDaily 每天在 hour:minute 执行一次（对齐 @Scheduled(cron)）。
func runDaily(ctx context.Context, hour, minute int, fn func()) {
	for {
		next := nextDaily(hour, minute)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			safeRun(fn)
		}
	}
}

// nextDaily 计算下一次执行时刻（今天未到则为今天，否则为明天）。
func nextDaily(hour, minute int) time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// ============================ ExamineOrder ============================

// cancelChargingOrder 定时检查预约超过 2 小时的驿享充电订单，取消并扣除定金（对齐 CancelChargingOrder）。
func cancelChargingOrder() {
	orders := repository.AllChargingByBooking()
	if len(orders) == 0 {
		return
	}
	list := examineOrderByTime(orders, 120, repository.OrderStateBooked)
	if len(list) == 0 {
		return
	}
	logger.Mylog.Info().Msgf("检查到超时的驿享充电订单: %v", list)
	for _, orderID := range list {
		order, err := repository.FindOrderByOrderid(orderID)
		if err != nil {
			continue
		}
		if CancelOrderFn != nil {
			CancelOrderFn(orderID, order.DecideAmount.String())
		}
	}
}

// cancelChargingOrderIn15Minutes 取消预约超过 15 分钟的驿享充电订单，金额置 0（对齐 CancelChargingOrderIn15Minutes）。
func cancelChargingOrderIn15Minutes() {
	now := time.Now()
	for _, o := range repository.AllChargingByBooking() {
		if o.State != repository.OrderStateBooked && o.State != repository.OrderStateBooking {
			continue
		}
		if o.ReserveTime == nil {
			continue
		}
		if now.Sub(*o.ReserveTime).Minutes() >= 15 {
			logger.Mylog.Info().Msgf("检查到超过15分钟的驿享充电订单，自动取消: %s", o.Orderid)
			if CancelOrderFn != nil {
				CancelOrderFn(o.Orderid, "0")
			}
		}
	}
}

// cancelTwoWheelChargingOrderIn15Minutes 取消超过 5 分钟的"二轮车充电 进行中"订单，金额置 0（对齐 CancelTwoWheelChargingOrderIn15Minutes）。
func cancelTwoWheelChargingOrderIn15Minutes() {
	now := time.Now()
	for _, o := range repository.OrdersByStateAndConsumption(repository.ConsumeTypeTwice, repository.OrderStateConduct) {
		if o.ReserveTime == nil {
			continue
		}
		if now.Sub(*o.ReserveTime).Minutes() >= 5 {
			logger.Mylog.Info().Msgf("检查到超过5分钟的二轮车充电订单，自动取消: %s", o.Orderid)
			if CancelOrderFn != nil {
				CancelOrderFn(o.Orderid, "0")
			}
		}
	}
}

// closeBalanceNotEnoughOrder 检测使用中订单余额是否足够支付，不够则断电并结束（对齐 CloseBalanceNotEnoughOrder）。
func closeBalanceNotEnoughOrder() {
	orders := repository.OrdersByStateAndConsumption(repository.ConsumeTypeYiXiang, repository.OrderStateUse)
	var closed []string
	for _, o := range orders {
		if GetOrderPayFn != nil && GetOrderPayFn(o.Orderid) == "微信支付分支付" {
			continue
		}
		user, err := repository.UserByOpenid(context.Background(), o.Openid)
		if err != nil {
			continue
		}
		totalAmount := user.Balans.Add(user.FreezeBalans)
		if totalAmount.GreaterThan(decimal.NewFromFloat(150)) {
			continue
		}
		rechargeAmount, _ := repository.ElectronicFeesByOrderid(o.Orderid)
		if totalAmount.Sub(rechargeAmount).LessThan(decimal.NewFromFloat(5)) {
			if ByOrderIdSendCloseEleAndMessageFn != nil {
				ByOrderIdSendCloseEleAndMessageFn(o.Orderid)
			}
			closed = append(closed, o.Orderid)
		}
	}
	if len(closed) > 0 {
		logger.Mylog.Info().Msgf("因余额不足,断电的订单: %v", closed)
	}
}

// examineOrderByTime 根据时间阈值筛选超时订单（对齐 ExamineOrderByTime）。
func examineOrderByTime(orders []model.OrderTbl, minutes int, state string) []string {
	now := time.Now()
	var list []string
	for _, o := range orders {
		var between time.Duration
		if o.ConsumptionType == repository.ConsumeTypeTwice {
			if o.ReserveTime == nil {
				continue
			}
			between = now.Sub(*o.ReserveTime)
		} else {
			if o.BookingStartTime == nil {
				logger.Mylog.Warn().Msgf("订单号:%s, 非预约订单", o.Orderid)
				continue
			}
			between = now.Sub(*o.BookingStartTime)
		}
		if between.Minutes() >= float64(minutes) && o.State == state {
			list = append(list, o.Orderid)
		}
	}
	return list
}

// ============================ RefundStatusSyncTask ============================

// syncRefundStatus 每 30 分钟同步一次退款状态（对齐 syncRefundStatus）。
func syncRefundStatus() {
	logger.Mylog.Info().Msg("开始执行退款状态同步任务")
	records := repository.PendingRefundRecords([]string{refundStatusApplying, refundStatusProcessing}, false)
	if len(records) == 0 {
		logger.Mylog.Info().Msg("没有需要同步状态的退款记录")
		return
	}
	logger.Mylog.Info().Msgf("发现%d条需要同步状态的退款记录", len(records))
	success, failed := 0, 0
	for _, rec := range records {
		if syncOneRefund(rec) {
			success++
		} else {
			failed++
		}
	}
	logger.Mylog.Info().Msgf("退款状态同步任务完成：成功%d条，失败%d条", success, failed)
}

// fullSyncRefundStatus 每天凌晨 2 点全量同步退款状态（对齐 fullSyncRefundStatus）。
func fullSyncRefundStatus() {
	logger.Mylog.Info().Msg("开始执行全量退款状态同步任务")
	records := repository.PendingRefundRecords(nil, true)
	if len(records) == 0 {
		logger.Mylog.Info().Msg("没有需要全量同步的退款记录")
		return
	}
	logger.Mylog.Info().Msgf("发现%d条需要全量同步的退款记录", len(records))
	success, failed := 0, 0
	for _, rec := range records {
		if syncOneRefund(rec) {
			success++
		} else {
			failed++
		}
	}
	logger.Mylog.Info().Msgf("全量退款状态同步任务完成：成功%d条，失败%d条", success, failed)
}

// syncOneRefund 同步单条退款记录状态，返回是否成功（无变化视为成功）。
func syncOneRefund(rec model.WithdrawalRecordTbl) bool {
	if RefundQueryFn == nil {
		return false
	}
	wxStatus, err := RefundQueryFn(rec.RefundOrderID)
	if err != nil {
		logger.Mylog.Error().Msgf("退款记录%s状态同步异常：%s", rec.RecordID, err.Error())
		return false
	}
	refundStatus := mapWxRefundStatus(wxStatus)
	if rec.WithdrawalStatus == refundStatus {
		return true
	}
	now := time.Now()
	updates := map[string]interface{}{
		"withdrawal_status": refundStatus,
		"update_time":       now,
	}
	if refundStatus == refundStatusSuccess {
		updates["process_time"] = now
		updates["refund_status_desc"] = "退款成功"
	}
	if repository.UpdateWithdrawalRecord(rec.RecordID, updates) > 0 {
		logger.Mylog.Info().Msgf("退款记录%s状态同步成功：%s -> %s", rec.RecordID, rec.WithdrawalStatus, refundStatus)
		return true
	}
	logger.Mylog.Warn().Msgf("退款记录%s状态更新失败", rec.RecordID)
	return false
}

// mapWxRefundStatus 映射微信退款状态到系统退款状态（对齐 mapWxRefundStatus）。
func mapWxRefundStatus(wxStatus string) string {
	switch strings.ToUpper(wxStatus) {
	case "SUCCESS":
		return refundStatusSuccess
	case "REFUNDCLOSE":
		return refundStatusClosed
	case "PROCESSING", "CHANGE":
		return refundStatusProcessing
	default:
		return refundStatusRejected
	}
}

// ============================ RenewalBooking ============================

// renewalBooking 每天 0 点重置所有用户预约次数为 3（对齐 RenewalBooking.updateDatabase）。
func renewalBooking() {
	date := time.Now().Format("2006-01-02")
	rows := repository.RenewalBookingAll(3, date)
	logger.Mylog.Info().Msgf("每日预约次数重置完成, 影响行数: %d, 日期: %s", rows, date)
}
