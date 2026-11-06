package service

import (
	"strconv"
	"strings"

	"caicai-go/logger"
	"caicai-go/repository"
)

// SettleTwiceOrderFn 二轮车结算函数，由 handler 注入（用于打破 service→handler 循环依赖）。
// 签名与 handler.settleTwiceOrder 一致：withGauge 表示结算时是否带关闸。
var SettleTwiceOrderFn func(orderID string, withGauge bool) bool

// FinishGunReturnOrderFn 充电枪归还（OpChargeGunBack）自动结束订单函数，由 handler 注入（打破循环依赖）。
var FinishGunReturnOrderFn func(orderID string, meter int32)

// StartOrderFn 开启订单（对齐 Java OrderServiceImpl.start），由 handler 注入。
// 供延迟任务分发器 parking_start_charge_queue 消费者调用。
var StartOrderFn func(orderID string) bool

// CloseOrderFn 结束订单（对齐 Java OrderServiceImpl.close），由 handler 注入。
// 供延迟任务分发器使用（如需在异步任务中结算驿享充电订单）。
var CloseOrderFn func(orderID string) bool

// CancelOrderFn 取消订单（对齐 Java OrderServiceImpl.cancelOrder），由 handler 注入。
// 供定时任务（ExamineOrder）调用。
var CancelOrderFn func(orderID, price string) bool

// ByOrderIdSendCloseEleAndMessageFn 断电并发送消息（对齐 Java ByOrderIdSendCloseEleAndMessage），由 handler 注入。
// 供定时任务（CloseBalanceNotEnoughOrder）调用。
var ByOrderIdSendCloseEleAndMessageFn func(orderID string) bool

// RefundQueryFn 查询微信退款状态（对齐 Java WxPayService.refundQuery），由 handler 注入。
// 返回微信退款状态：SUCCESS / PROCESSING / CHANGE / ABNORMAL / CLOSED / REFUNDCLOSE。
var RefundQueryFn func(outRefundNo string) (string, error)

// GetOrderPayFn 判断订单支付方式（对齐 Java OrderServiceImpl.getOrderPay），由 handler 注入。
// 返回 "微信支付分支付" 或 "余额支付"。
var GetOrderPayFn func(orderID string) string

// OfflineSettleOrderFn 设备离线结算订单函数（按最后一次上报电量计费并扣款），由 handler 注入。
// 供设备心跳超时（handleDeviceOffline）对驿享充电/邻享订单调用。
var OfflineSettleOrderFn func(orderID string)

// CloseGaugeByOrder 关闭订单对应插座的继电器（对齐 Java CloseGAUGE）。
func CloseGaugeByOrder(orderID string) {
	receptacle := repository.ReceptacleByOrderid(orderID)
	if receptacle == nil {
		return
	}
	product := repository.ProductByPid(receptacle.Pid)
	if product == nil {
		return
	}
	rid, _ := strconv.Atoi(strings.Split(receptacle.Name, "号")[0])
	logger.Mylog.Info().Msgf("关闸(按订单): order=%s imei=%s rid=%d", orderID, product.Imei, rid)
	CloseGauge(product.Imei, rid)
}

// GaugeFull 二轮车充满自停检测（对齐 Java MessageListeningServiceImpl.GaugeFull）。
// 功率连续 3 次低于阈值且订单仍在使用中/进行中时，异步关闸并结算（不带关闸）。
func GaugeFull(orderID string) {
	count, reached := repository.GaugeFullStep(orderID)
	if !reached {
		return
	}
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || (order.State != repository.OrderStateUse && order.State != repository.OrderStateConduct) {
		return
	}
	logger.Mylog.Info().Msgf("二轮车充满自停触发: order=%s count=%d", orderID, count)
	go CloseGaugeByOrder(orderID)
	if SettleTwiceOrderFn != nil {
		SettleTwiceOrderFn(orderID, false)
	}
}
