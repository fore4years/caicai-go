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
