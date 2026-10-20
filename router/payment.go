package router

import (
	"github.com/gin-gonic/gin"

	"caicai-go/handler"
)

// registerPayment 注册微信支付（WeChatPaymentController）与微信支付分（WxPayController）路由。
func registerPayment(r *gin.Engine) {
	payment := new(handler.WeChatPaymentController)
	wxPay := new(handler.WxPayController)

	pay := r.Group("/pay")
	{
		pay.Any("/payment", payment.Payment)
		pay.POST("/recharge", payment.Recharge)
		pay.Any("/payCallback", payment.PayCallback)
	}

	wx := r.Group("/wx")
	{
		wx.Any("/wechat/callback", wxPay.OnWxCallbackNotify)      // 支付分回调（对齐 Java /wechat/callback）
		wx.Any("/wechat/refund/notify", wxPay.OnWxCallbackNotify) // 退款回调（对齐 Java，转调 wechatCallback）
		wx.POST("/createPayOrder", wxPay.CreatePayOrder)          // 创建支付分订单
		wx.Any("/queryWxOrder", wxPay.QueryWxOrder)
		wx.Any("/cancelWxOrder", wxPay.CancelWxOrder)
		wx.Any("/completeWxOrder", wxPay.CompleteWxOrder)
		wx.GET("/refunds", wxPay.Refunds)
		wx.GET("/refund-query", wxPay.RefundQuery) // 退款状态查询
		wx.POST("/subtractDefaultAmount", wxPay.SubtractDefaultAmount)
	}
}
