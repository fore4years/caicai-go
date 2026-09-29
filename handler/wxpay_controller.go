package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
)

// ============================================================================
// 微信支付 / 微信支付分控制器，对齐 Java：
//   - controller/WeChatPaymentController.java（base /pay）
//   - controller/WxPayController.java（base /wx）
//
// 微信支付 v3 的真实 HTTP 调用（下单 / 查询 / 关单 / 完结 / 退款 / 回调验签解密）在 Go 里
// 没有对等 SDK 基础设施，因此沿用 handler/wechat_pay.go 的桩风格：复用其中的
// completeWxOrder / cancelWxOrder / subtractWxDefaultAmount / refundWxOrder / queryWxOrderState。
// DB 层的余额写入（recharge）与金额计算（completeWxOrder 的电费明细）忠实还原。
// ============================================================================

// 对齐 Java wechat.WxPayConfig 的微信支付 v3 常量。真实 HTTP 调用被 stub，
// 此处仅用于构造/记录请求参数（保留常量以便后续接入真实 SDK 时直接使用）。
const (
	wxPayAppID       = "wx4fc1474ea10f5d5e"
	wxPayMerchantID  = "1662744190"
	wxPayMchSerialNo = "5248C4949E9441F47DDABD76C29E9C65041566CE"
	wxPayAPIv3Key    = "Wisdom028qqcomlanlan818331125111"
	wxPayServiceID   = "00003004000000170685804269033905"
	wxPayServiceName = "成都七彩云创信息技术有限公司"
	wxPayRefundURL   = "https://api.mch.weixin.qq.com/v3/refund/domestic/refunds"
	wxPayBaseURL     = "https://api.mch.weixin.qq.com/v3/payscore/serviceorder"
	wxPayNotifyURL   = "https://www.caiparking.com/pay/payCallback"
)

// ============================ 请求 / 响应 DTO ============================

// wechatPayRequest 对齐 Java model.WeChatPayBen（Integer totalfee, String tradeno, String openid）。
type wechatPayRequest struct {
	Totalfee int    `json:"totalfee"`
	Tradeno  string `json:"tradeno"`
	Openid   string `json:"openid"`
}

// rechargeRequest 对齐 Java model.UserBean 中充值接口用到的字段（openid, Double balans）。
type rechargeRequest struct {
	Openid string          `json:"openid"`
	Balans decimal.Decimal `json:"balans"`
}

// prepayResponse 对应 Java 微信 SDK PrepayWithRequestPaymentResponse（jsapi 下单应答）。
// 真实字段为 prepay_id；桩实现返回占位。
type prepayResponse struct {
	PrepayID   string `json:"prepayId,omitempty"`
	OutTradeNo string `json:"outTradeNo,omitempty"`
}

// wxQueryOrderRes 对齐 Java domain.wechat.WxQueryOrderRes（支付分查询订单响应）。
type wxQueryOrderRes struct {
	Appid               string `json:"appid,omitempty"`
	Mchid               string `json:"mchid,omitempty"`
	ServiceID           string `json:"service_id,omitempty"`
	OutOrderNo          string `json:"out_order_no,omitempty"`
	ServiceIntroduction string `json:"service_introduction,omitempty"`
	State               string `json:"state,omitempty"`
	StateDescription    string `json:"state_description,omitempty"`
	TotalAmount         int    `json:"total_amount,omitempty"`
	OrderID             string `json:"order_id,omitempty"`
	NeedCollection      bool   `json:"need_collection,omitempty"`
}

// wxCancelOrderRes 对齐 Java domain.wechat.WxCancelOrderRes（支付分取消订单响应）。
type wxCancelOrderRes struct {
	Appid      string `json:"appid,omitempty"`
	Mchid      string `json:"mchid,omitempty"`
	OutOrderNo string `json:"out_order_no,omitempty"`
	ServiceID  string `json:"service_id,omitempty"`
	OrderID    string `json:"order_id,omitempty"`
}

// wxCompleteOrderRes 对齐 Java domain.wechat.WxCompleteOrderRes（支付分完结订单响应）。
type wxCompleteOrderRes struct {
	Appid               string `json:"appid,omitempty"`
	Mchid               string `json:"mchid,omitempty"`
	OutOrderNo          string `json:"out_order_no,omitempty"`
	ServiceID           string `json:"service_id,omitempty"`
	ServiceIntroduction string `json:"service_introduction,omitempty"`
	State               string `json:"state,omitempty"`
	StateDescription    string `json:"state_description,omitempty"`
	TotalAmount         int    `json:"total_amount,omitempty"`
	OrderID             string `json:"order_id,omitempty"`
	NeedCollection      bool   `json:"need_collection,omitempty"`
}

// refundResult 对齐 Java model.wechat.RefundResult（退款响应）。
type refundResult struct {
	RefundID            string           `json:"refund_id,omitempty"`
	TransactionID       string           `json:"transaction_id,omitempty"`
	OutTradeNo          string           `json:"out_trade_no,omitempty"`
	OutRefundNo         string           `json:"out_refund_no,omitempty"`
	Channel             string           `json:"channel,omitempty"`
	UserReceivedAccount string           `json:"user_received_account,omitempty"`
	Status              string           `json:"status,omitempty"`
	FundsAccount        string           `json:"funds_account,omitempty"`
	Amount              *refundAmountDTO `json:"amount,omitempty"`
}

// refundAmountDTO 对齐 Java RefundResult.amountDTO。
type refundAmountDTO struct {
	Total            int    `json:"total,omitempty"`
	Refund           int    `json:"refund,omitempty"`
	PayerTotal       int    `json:"payer_total,omitempty"`
	PayerRefund      int    `json:"payer_refund,omitempty"`
	SettlementRefund int    `json:"settlement_refund,omitempty"`
	SettlementTotal  int    `json:"settlement_total,omitempty"`
	DiscountRefund   int    `json:"discount_refund,omitempty"`
	Currency         string `json:"currency,omitempty"`
}

// ============================ 控制器 ============================

// WeChatPaymentController 对齐 Java controller/WeChatPaymentController（base /pay）。
type WeChatPaymentController struct{}

// WxPayController 对齐 Java controller/WxPayController（base /wx）。
type WxPayController struct{}

// formOrQueryInt 读取 POST 表单参数，缺失时回退到查询参数（对齐 Spring 未标注注解的
// String/int 方法参数：既可从 query 也可从 form 绑定）。
func formOrQueryInt(c *gin.Context, key string) int {
	if v := c.PostForm(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return queryInt(c, key, 0)
}

// Payment 微信支付下单，对齐 Java getpayment -> WeChatPayServiceApiV3.preOrder。
func (w *WeChatPaymentController) Payment(c *gin.Context) {
	var req wechatPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultError(1, "参数错误"))
		return
	}

	// 对齐 Java preOrder：金额单位分 = totalfee * 100，币种 = tradeno，商品描述「余额充值」，
	// 回调地址 wxPayNotifyURL，商户订单号 outTradeNo = UUID（IdUtil.simpleUUID）。
	totalFen := req.Totalfee * 100
	outTradeNo := simpleUUID()
	logger.Mylog.Info().Msgf("[wxpay stub] payment 下单: totalFen=%d tradeno=%s openid=%s outTradeNo=%s",
		totalFen, req.Tradeno, req.Openid, outTradeNo)

	// 真实实现调用 JsapiServiceExtension.prepayWithRequestPayment(...) 返回
	// PrepayWithRequestPaymentResponse（prepayId）。此处为占位。
	c.JSON(http.StatusOK, ResultSuccess(prepayResponse{OutTradeNo: outTradeNo}))
}

// Recharge 余额充值，对齐 Java recharge -> UserServiceImpl.updateRecharge。
func (w *WeChatPaymentController) Recharge(c *gin.Context) {
	var req rechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultError(1, "参数错误"))
		return
	}

	// 对齐 Java updateRecharge：UPDATE user_tbl SET balans = balans + ? WHERE openid = ?。
	var user model.UserTbl
	if conf.Db.Where("openid = ?", req.Openid).First(&user).Error != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	newBalans := user.Balans.Add(req.Balans)
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", req.Openid).Update("balans", newBalans)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// PayCallback 微信支付 v3 回调，对齐 Java weChatPayCallback。
func (w *WeChatPaymentController) PayCallback(c *gin.Context) {
	// Java 该方法体中的验签 / 解密 / 订单状态更新全部被注释掉，仅打印日志后 void 返回。
	// 因此这里仅解析 v3 回调头做日志记录，不验签、不落库，保持与 Java 一致。
	logger.Mylog.Info().Msg("支付回调通知成功啦")

	serial := c.GetHeader("Wechatpay-Serial")
	nonce := c.GetHeader("Wechatpay-Nonce")
	signature := c.GetHeader("Wechatpay-Signature")
	timestamp := c.GetHeader("Wechatpay-Timestamp")
	logger.Mylog.Info().Msgf("[wxpay stub] payCallback 验签头: serial=%s nonce=%s signature=%s timestamp=%s",
		serial, nonce, signature, timestamp)

	// Java 返回 void（HTTP 200 空 body）。真实接入后此处应验签、解密 resource 并更新
	// 充值订单状态（recharge_order_tbl.order_status -> 已支付）与用户余额，Java 中已注释掉。
	c.Status(http.StatusOK)
}

// OnWxCallbackNotify 微信支付分回调，对齐 Java WxPayController.onWxCallbackNotify。
func (w *WxPayController) OnWxCallbackNotify(c *gin.Context) {
	// Java 该方法与 WxPayImp.onWxCallbackNotify 均仅打印日志 / return null，无 DB 写入、
	// 无订单状态流转。这里保持一致，仅返回 200 空 body。
	logger.Mylog.Info().Msg("\n支付分回调通知成功啦！")
	c.Status(http.StatusOK)
}

// QueryWxOrder 查询支付分订单，对齐 Java onQueryWxOrder。
func (w *WxPayController) QueryWxOrder(c *gin.Context) {
	outOrderNo := c.Query("outOrderNo")
	logger.Mylog.Info().Msgf("支付分:使用%s查询订单", outOrderNo)

	// 对齐 Java WxPayImp.onQueryWxOrder：GET wxPayBaseURL + "?out_order_no=..&service_id=..&appid=.."。
	// 真实 HTTP 查询被 stub；state 复用 queryWxOrderState（桩恒返回 CANCEL）。
	state := queryWxOrderState(outOrderNo)
	res := wxQueryOrderRes{
		Appid:      wxPayAppID,
		ServiceID:  wxPayServiceID,
		OutOrderNo: outOrderNo,
		State:      state,
	}
	logger.Mylog.Info().Msgf("支付分:查询返回%+v", res)
	c.JSON(http.StatusOK, ResultSuccess(res))
}

// CancelWxOrder 取消支付分订单，对齐 Java onCancelWxOrder。
func (w *WxPayController) CancelWxOrder(c *gin.Context) {
	outOrderNo := c.Query("outOrderNo")
	// 对齐 Java WxPayImp.onCancelWxOrder：POST wxPayBaseURL+"/"+outOrderNo+"/cancel"，
	// body {appid, service_id, reason:"无需付款"}。真实 HTTP 被 stub。
	cancelWxOrder(outOrderNo)
	c.JSON(http.StatusOK, ResultSuccess(wxCancelOrderRes{OutOrderNo: outOrderNo}))
}

// CompleteWxOrder 完结支付分订单，对齐 Java onCompleteWxOrder。
func (w *WxPayController) CompleteWxOrder(c *gin.Context) {
	outOrderNo := c.Query("outOrderNo")
	address := c.Query("address")

	// 对齐 Java WxPayImp.onCompleteWxOrder：读取订单电费明细构造 post_payments 描述
	// （getElectronicByOrderid -> totalFee / serviceFee，getFeesByOrderid -> fees）。
	totalFee, serviceFee := electronicFeesByOrderid(outOrderNo)
	fees := totalFee.Add(serviceFee).Round(2)
	servicePart := fees.Sub(totalFee).Round(2)
	desc := "电费" + totalFee.String() + "元,服务费" + servicePart.String() + "元"
	logger.Mylog.Info().Msgf("订单完成: outOrderNo=%s address=%s 费用明细=%s", outOrderNo, address, desc)

	// 控制器将 amount 硬编码为 1（对齐 Java onCompleteWxOrder(outOrderNo, address, 1)）。
	// POST wxPayBaseURL+"/"+outOrderNo+"/complete" 被 stub。
	completeWxOrder(outOrderNo, 1)
	c.JSON(http.StatusOK, ResultSuccess(wxCompleteOrderRes{OutOrderNo: outOrderNo}))
}

// Refunds 支付分退款，对齐 Java refunds。
func (w *WxPayController) Refunds(c *gin.Context) {
	transactionID := c.Query("transactionId")
	outRefundNo := c.Query("outRefundNo")
	refund := queryInt(c, "refund", 0)
	total := queryInt(c, "total", 0)

	// 对齐 Java WxPayImp.refunds：body 为 {transaction_id, out_refund_no, amount{refund,total,currency:CNY}}，
	// POST 到 wxPayRefundURL。真实退款 HTTP 调用被 stub。
	refundWxOrder(transactionID, outRefundNo, refund, total)

	// 真实实现返回微信退款结果 RefundResult；此处为占位空对象。
	c.JSON(http.StatusOK, ResultSuccess(refundResult{}))
}

// SubtractDefaultAmount 取消订单扣除违约金/定金，对齐 Java subtractDefaultAmount。
func (w *WxPayController) SubtractDefaultAmount(c *gin.Context) {
	outOrderNo := c.Query("outOrderNo")
	if outOrderNo == "" {
		outOrderNo = c.PostForm("outOrderNo")
	}
	amount := formOrQueryInt(c, "amount")

	// 对齐 Java WxPayImp.subtractDefaultAmount：POST wxPayBaseURL+"/"+outOrderNo+"/complete"，
	// total_amount = amount（分），post_payments 描述「因当前时间距离预约时间较近,系统已自动按照扣费率扣除定金」。
	// 真实 HTTP 调用被 stub。
	subtractWxDefaultAmount(outOrderNo, amount)
	c.JSON(http.StatusOK, ResultSuccess(wxCompleteOrderRes{OutOrderNo: outOrderNo}))
}

// ============================ 路由注册 ============================

// RegisterPayment 注册微信支付相关路由（对齐 Java WeChatPaymentController / WxPayController）。
// 注意：本函数不会自动接入，需在 router 初始化处调用（任务约定不改动既有 router/router.go）。
func RegisterPayment(r *gin.Engine) {
	payment := new(WeChatPaymentController)
	wxPay := new(WxPayController)

	pay := r.Group("/pay")
	{
		pay.Any("/payment", payment.Payment)         // Java @RequestMapping("payment")
		pay.POST("/recharge", payment.Recharge)      // Java @PostMapping("recharge")
		pay.Any("/payCallback", payment.PayCallback) // Java @RequestMapping("payCallback")
	}

	wx := r.Group("/wx")
	{
		wx.Any("/onWxCallbackNotify", wxPay.OnWxCallbackNotify)        // Java @RequestMapping
		wx.Any("/queryWxOrder", wxPay.QueryWxOrder)                    // Java @RequestMapping
		wx.Any("/cancelWxOrder", wxPay.CancelWxOrder)                  // Java @RequestMapping
		wx.Any("/completeWxOrder", wxPay.CompleteWxOrder)              // Java @RequestMapping
		wx.GET("/refunds", wxPay.Refunds)                              // Java @GetMapping
		wx.POST("/subtractDefaultAmount", wxPay.SubtractDefaultAmount) // Java @PostMapping
	}
}
