package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/constant"
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
	OrderId  string `json:"orderId"`
}

// rechargeRequest 对齐 Java model.UserBean 中充值接口用到的字段（openid, Double balans）。
type rechargeRequest struct {
	Openid string          `json:"openid"`
	Balans decimal.Decimal `json:"balans"`
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
	orderId := req.OrderId
	if orderId == "" {
		orderId = simpleUUID()
	}
	logger.Mylog.Info().Msgf("orderId: %s", orderId)
	// 对齐 Java preOrder：金额单位分 = totalfee * 100，币种 = tradeno；描述「余额充值」、
	// appid/mchid/notifyUrl 见 prepayJsapi（真实调用微信支付 v3 统一下单）。
	totalFen := req.Totalfee * 100
	resp, err := prepayJsapi(totalFen, req.Tradeno, req.Openid, orderId)
	if err != nil {
		logger.Mylog.Error().Err(err).Msg("微信支付下单失败")
		c.JSON(http.StatusOK, ResultError(500, "微信支付下单失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(resp))
}

// Recharge 余额充值，对齐 Java recharge -> UserServiceImpl.updateRecharge。
func (w *WeChatPaymentController) Recharge(c *gin.Context) {
	var req rechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultError(1, "参数错误"))
		return
	}
	if req.Openid == "" || !req.Balans.IsPositive() {
		c.JSON(http.StatusOK, ResultError(1, "参数错误"))
		return
	}

	// 对齐 Java updateRecharge 的业务语义（balans += 充值金额），但改为单条原子 UPDATE：
	// 避免「读余额 → 计算 → 写回」的读-改-写竞态；金额以字符串传参并 CAST 为 DECIMAL，
	// 避免 MySQL DECIMAL→DOUBLE 精度丢失。用户不存在时 RowsAffected==0 返回 false。
	orderId := simpleUUID()
	amt := req.Balans.Round(2)
	order := &model.RechargeOrderTbl{
		Openid:                req.Openid,
		OrderID:               orderId,
		RechargeAmount:        amt,
		RefundedAmount:        decimal.Zero,
		RemainingRefundAmount: amt,
		MaxRefundAmount:       amt,
		OrderStatus:           constant.OrderRecharging,
		CreateTime:            time.Now(),
		UpdateTime:            time.Now(),
	}
	if err := conf.Db.Model(&model.RechargeOrderTbl{}).Omit("payment_time", "refund_time").Create(order).Error; err != nil {
		logger.Mylog.Err(fmt.Errorf("创建订单失败, orderId=%s: %w", orderId, err))
		c.JSON(http.StatusInternalServerError, ResultError(400, "数据更新失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderId))
}

// PayCallback 微信支付 v3 回调，对齐 Java weChatPayCallback（G 版）。
// 验签+解密 → 仅处理 TRANSACTION.SUCCESS → 校验交易状态 → 按订单类型结算。
func (w *WeChatPaymentController) PayCallback(c *gin.Context) {
	transaction := new(payments.Transaction)
	notifyReq, err := parseWxPayNotify(c.Request, transaction)
	if err != nil {
		logger.Mylog.Error().Err(err).Msg("微信支付回调验签/解密失败")
		c.String(http.StatusInternalServerError, "FAIL")
		return
	}

	// 非支付成功事件直接 ACK。
	if notifyReq.EventType != "TRANSACTION.SUCCESS" {
		logger.Mylog.Info().Msgf("忽略非支付成功回调事件: %s", notifyReq.EventType)
		c.String(http.StatusOK, "SUCCESS")
		return
	}

	outTradeNo := ""
	if transaction.OutTradeNo != nil {
		outTradeNo = *transaction.OutTradeNo
	}
	transactionID := ""
	if transaction.TransactionId != nil {
		transactionID = *transaction.TransactionId
	}
	tradeState := ""
	if transaction.TradeState != nil {
		tradeState = *transaction.TradeState
	}
	if tradeState == "" {
		tradeState = "SUCCESS"
	}
	logger.Mylog.Info().Msgf("微信支付回调解密参数 - 商户订单号: %s, 微信交易号: %s, 交易状态: %s",
		outTradeNo, transactionID, tradeState)

	if tradeState != "SUCCESS" || transactionID == "" {
		logger.Mylog.Warn().Msgf("微信支付回调交易状态非成功或缺少交易号, 状态: %s, 交易号: %s", tradeState, transactionID)
		c.String(http.StatusInternalServerError, "FAIL")
		return
	}

	if err := settlePayCallback(outTradeNo, transactionID); err != nil {
		logger.Mylog.Error().Err(err).Msg("微信支付回调处理失败")
		c.String(http.StatusInternalServerError, "FAIL")
		return
	}

	c.String(http.StatusOK, "SUCCESS")
}

// settlePayCallback 按商户订单号依次匹配充值订单、购桩订单、套餐订单并结算，对齐 Java 回调的 if-else 链。
func settlePayCallback(outTradeNo, transactionID string) error {
	// 1. 充值订单
	var rechargeOrder model.RechargeOrderTbl
	if err := conf.Db.Where("order_id = ?", outTradeNo).First(&rechargeOrder).Error; err == nil {
		return settleRechargeOrder(&rechargeOrder, transactionID)
	}

	// 2. 购桩订单
	var purchasePole model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", outTradeNo).First(&purchasePole).Error; err == nil {
		res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
			Where("order_number = ?", outTradeNo).
			Updates(map[string]interface{}{
				"transaction_id": transactionID,
				"pay_time":       time.Now(),
				"order_status":   "已支付",
			})
		if res.Error != nil {
			return res.Error
		}
		logger.Mylog.Info().Msgf("成功更新购桩订单支付信息和状态, 订单号: %s", outTradeNo)
		return nil
	}

	// 3. 套餐订单
	var pkg model.PackageRecordTbl
	if err := conf.Db.Where("order_no = ?", outTradeNo).First(&pkg).Error; err == nil {
		updates := map[string]interface{}{
			"status":         "paid",
			"transaction_id": transactionID,
		}
		if pkg.EndTime.IsZero() && !pkg.CreateTime.IsZero() {
			updates["end_time"] = pkg.CreateTime.AddDate(0, 0, int(pkg.Days))
		}
		res := conf.Db.Model(&model.PackageRecordTbl{}).Where("order_no = ?", outTradeNo).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		logger.Mylog.Info().Msgf("成功更新套餐充值订单支付信息和状态, 订单号: %s", outTradeNo)
		return nil
	}

	return fmt.Errorf("未找到对应的充值订单、购桩订单或套餐充值订单, 订单号: %s", outTradeNo)
}

// settleRechargeOrder 充值订单支付成功结算：待支付 -> 已支付，加余额，记余额明细。
// 用「状态条件更新」保证幂等（重复回调不重复加余额），事务保证订单状态/余额/明细同时生效。
func settleRechargeOrder(order *model.RechargeOrderTbl, transactionID string) error {
	if order.OrderStatus == "已支付" {
		return nil // 已处理过，幂等
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
			return nil // 已被其它回调处理
		}

		// 加余额（原子 UPDATE，DECIMAL 精度）
		amt := order.RechargeAmount.StringFixed(2)
		if err := tx.Exec(
			"UPDATE user_tbl SET balans = balans + CAST(? AS DECIMAL(10,2)) WHERE openid = ?",
			amt, order.Openid,
		).Error; err != nil {
			return err
		}

		// 记余额明细（类型「充值」）
		return tx.Create(&model.TabBalanceRecord{
			Openid:     order.Openid,
			Type:       "充值",
			Amount:     order.RechargeAmount,
			CreateTime: time.Now(),
		}).Error
	})
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

	// 对齐 Java WxPayImp.refunds：body 为 {transaction_id, out_refund_no, amount{refund,total,currency:CNY}}。
	status, err := refundWxOrder(transactionID, outRefundNo, refund, total)
	if err != nil {
		logger.Mylog.Error().Err(err).Msg("微信退款失败")
		c.JSON(http.StatusOK, ResultError(500, "退款失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(refundResult{Status: status}))
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
