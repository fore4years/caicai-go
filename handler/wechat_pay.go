package handler

import (
	"caicai-go/logger"
)

// ============================================================================
// 微信支付（微信支付分）客户端抽象，对齐 Java WxPayService（实现类 WxPayImp，约 1090 行）。
//
// 当前为桩实现：queryWxOrderState 恒返回 "CANCEL"（语义为「未使用微信支付分」，
// 上层将回退到余额支付），其余方法仅记录日志。接入真实微信支付 v3
// （wechatpay-go SDK + 商户证书/密钥/序列号，配置项见 Java application.yml 的 wxpay 段）
// 时，仅需在此文件补齐 HTTP 调用与验签，不影响上层订单业务逻辑。
// ============================================================================

// 微信支付分订单状态。
const (
	wxPayStateCancel = "CANCEL"
	wxPayStateDone   = "DONE"
	wxPayStateDoing  = "DOING"
	wxPayStateRevoke = "REVOKED"
)

func logWxPay(method string, args ...interface{}) {
	logger.Mylog.Info().Msgf("[wxpay stub] %s %v", method, args)
}

// createPayScoreOrder 创建微信支付分订单（对齐 WxPayService.createPayOrder）。
// 返回给前端的 OpenWeChat 拉起参数对象（appId/timeStamp/nonceStr/package/signType/paySign）。
func createPayScoreOrder(orderID string, needProfitSharing bool, appID string) map[string]interface{} {
	logWxPay("createPayScoreOrder", orderID, needProfitSharing, appID)
	return map[string]interface{}{}
}

// queryWxOrderState 查询微信支付分订单状态（对齐 WxPayService.onQueryWxOrder）。
// 返回 DONE / DOING / REVOKED / CANCEL。桩实现恒返回 CANCEL（未使用微信支付分）。
func queryWxOrderState(orderID string) string {
	logWxPay("queryWxOrderState", orderID)
	return wxPayStateCancel
}

// completeWxOrder 完成微信支付分订单扣款（对齐 onCompleteWxOrder / onCompleteWxLockOrder /
// onCompleteWxPrivateOrder / onCompleteWxTwiceOrder，amount 单位：分）。
func completeWxOrder(orderID string, amount int) {
	logWxPay("completeWxOrder", orderID, amount)
}

// cancelWxOrder 取消微信支付分订单（对齐 WxPayService.onCancelWxOrder）。
func cancelWxOrder(orderID string) {
	logWxPay("cancelWxOrder", orderID)
}

// subtractWxDefaultAmount 扣除违约金/定金（对齐 WxPayService.subtractDefaultAmount，单位：分）。
func subtractWxDefaultAmount(orderID string, amount int) {
	logWxPay("subtractWxDefaultAmount", orderID, amount)
}

// refundWxOrder 退款（对齐 WxPayService.refunds）。
func refundWxOrder(transactionID, orderID string, refund, total int) {
	logWxPay("refundWxOrder", transactionID, orderID, refund, total)
}
