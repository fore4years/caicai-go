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
		wx.Any("/onWxCallbackNotify", wxPay.OnWxCallbackNotify)
		wx.Any("/queryWxOrder", wxPay.QueryWxOrder)
		wx.Any("/cancelWxOrder", wxPay.CancelWxOrder)
		wx.Any("/completeWxOrder", wxPay.CompleteWxOrder)
		wx.GET("/refunds", wxPay.Refunds)
		wx.POST("/subtractDefaultAmount", wxPay.SubtractDefaultAmount)
	}
}
