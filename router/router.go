package router

import (
	"github.com/gin-gonic/gin"

	"caicai-go/handler"
)

func RouterInit(r *gin.Engine) {
	userController := new(handler.UserController)
	shouyeController := new(handler.ShouyeController)
	serviceNumberController := new(handler.ServiceNumberController)
	wxFujinController := new(handler.WxFujinController)
	messageController := new(handler.MessageController)
	yezhuController := new(handler.YezhuController)
	yunController := new(handler.YunController)
	wodeController := new(handler.WodeController)
	weChatController := new(handler.WeChatController)
	orderController := new(handler.OrderController)

	// 对齐 Java WxShouyeCon 的 /wx/* 端点
	wx := r.Group("/wx")
	{
		wx.POST("/login", userController.WxLogin)
		wx.POST("/login_owner", userController.LoginOwner)
		wx.POST("/getPhoneNumber", userController.GetPhoneNumber)
		wx.POST("/getOwnerPhoneNumber", userController.GetOwnerPhoneNumber)
		wx.POST("/submit", userController.Submit)

		// 首页相关端点（Java 使用 @RequestMapping 未限定 method，此处用 Any 对齐）
		wx.Any("/getMessage", shouyeController.GetMessage)
		wx.Any("/getPlaces", shouyeController.GetPlaces)
		wx.Any("/saoma", shouyeController.Saoma)
		wx.Any("/saomaForCharge", shouyeController.SaomaForCharge)
		wx.Any("/hasOrder", shouyeController.HasOrder)
		wx.Any("/hasChargeOrder", shouyeController.HasChargeOrder)

		// 服务号 / 附近 / 消息
		wx.GET("/checkSignature", serviceNumberController.CheckSignature)
		wx.Any("/getLockid", wxFujinController.GetLockid)
		wx.Any("/getMsgStatus", messageController.GetMsgStatus)
		wx.Any("/updateMsgStatus", messageController.UpdateMsgStatus)
		wx.Any("/getmsgs", messageController.GetMsgs)
	}

	// 业主侧
	chezhu := r.Group("/chezhu")
	{
		chezhu.Any("/zhuce", yezhuController.Zhuce)
		chezhu.Any("/getLockList", yezhuController.GetLockList)
	}

	// 我的（订单）
	wode := r.Group("/wx/wode")
	{
		wode.Any("/getOrders", wodeController.GetOrders)
		wode.Any("/timeDifference", wodeController.TimeDifference)
		wode.Any("/getOrderXq", wodeController.GetOrderXq)
		wode.Any("/searchmsg", wodeController.Searchmsg)
	}

	// 微信登录/注册（对应 Java WeChatController）
	wechat := r.Group("/wechat")
	{
		wechat.GET("/getPhone", weChatController.GetPhone)
		wechat.POST("/register", weChatController.Register)
		wechat.POST("/om/register", weChatController.OmRegister)
		wechat.GET("/login", weChatController.Login)
		wechat.GET("/om/login", weChatController.OmLogin)
		wechat.POST("/yiParL/register", weChatController.YiParLRegister)
		wechat.GET("/yiParL/login", weChatController.YiParLLogin)
	}

	// 设备数据回调
	r.Any("/yun", yunController.Yun)

	// 订单 / 用户信息（鉴权由全局 Auth 中间件统一处理）
	order := r.Group("/order")
	{
		order.GET("/getNotFinish", orderController.GetNotFinish)
	}
	user := r.Group("/user")
	{
		user.GET("/getUserInfo", orderController.GetUserInfo)
	}
}
