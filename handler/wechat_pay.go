package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wechatpay-apiv3/wechatpay-go/core"

	"caicai-go/logger"
	"caicai-go/repository"
)

// ============================================================================
// 微信支付分（WeChat Pay Score）客户端，对齐 Java WxPayService / WxPayImp。
// 基于官方 wechatpay-go SDK 的 core.Client（复用 handler/wechatpay_client.go 的
// getWxPayClient），按 /v3/payscore/serviceorder 低层 API 直接下发/查询/取消/完结。
//
// 微信支付分订单状态：
//   CREATED：商户已创建服务订单；
//   DOING：服务订单进行中；
//   DONE：服务订单完成；
//   REVOKED：商户取消服务订单；
//   EXPIRED：服务订单已失效。
// ============================================================================

// 微信支付分订单状态（对齐 Java WxQueryOrderRes.state 的枚举，CANCEL 为「未使用支付分」哨兵值）。
const (
	wxPayStateCancel = "CANCEL"
	wxPayStateDone   = "DONE"
	wxPayStateDoing  = "DOING"
	wxPayStateRevoke = "REVOKED"
)

// 微信支付分创建订单的回调通知地址（notify-domain + /wx/wechat/callback，从 conf.yaml 读取，init 中赋值）。
var wxPayScoreNotifyURL string

// ============================ 响应 DTO（前端拉起支付参数） ============================

// OpenWeChat 对齐 Java domain.OpenWeChat（创建支付分订单返回给前端的拉起支付参数）。
type OpenWeChat struct {
	BusinessType string           `json:"businessType"`
	OutOrderNo   string           `json:"outOrderNo"`
	ExtraData    *openWeChatExtra `json:"extraData"`
}

// openWeChatExtra 对齐 Java OpenWeChat.ExtraDataDTO。
type openWeChatExtra struct {
	MchID     string `json:"mch_id"`
	Package   string `json:"package"`
	Timestamp string `json:"timestamp"`
	NonceStr  string `json:"nonce_str"`
	SignType  string `json:"sign_type"`
	Sign      string `json:"sign"`
}

// ============================ 请求 DTO ============================

// payScoreRequest 对齐 Java domain.wechat.response.PayScoreResponse（创建支付分订单请求）。
type payScoreRequest struct {
	OutOrderNo          string                `json:"out_order_no"`
	Appid               string                `json:"appid"`
	ServiceID           string                `json:"service_id"`
	ServiceIntroduction string                `json:"service_introduction"`
	PostPayments        []payScorePostPayment `json:"post_payments"`
	TimeRange           payScoreTimeRange     `json:"time_range"`
	RiskFund            payScoreRiskFund      `json:"risk_fund"`
	NotifyURL           string                `json:"notify_url"`
	NeedUserConfirm     bool                  `json:"need_user_confirm"`
	ProfitSharing       bool                  `json:"profit_sharing"`
}

type payScorePostPayment struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

type payScoreTimeRange struct {
	StartTime string `json:"start_time"`
}

type payScoreRiskFund struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
}

// wxCreateOrderRes 对齐 Java domain.wechat.result.WxCreateOrderRes（创建订单响应，含 package 字段）。
type wxCreateOrderRes struct {
	Appid      string `json:"appid"`
	Mchid      string `json:"mchid"`
	OutOrderNo string `json:"out_order_no"`
	ServiceID  string `json:"service_id"`
	State      string `json:"state"`
	Package    string `json:"package"`
}

// wxCompleteOrder 对齐 Java domain.wechat.WxCompleteOrder（完结支付分订单请求）。
type wxCompleteOrder struct {
	Appid         string                  `json:"appid"`
	ServiceID     string                  `json:"service_id"`
	PostPayments  []wxCompletePostPayment `json:"post_payments"`
	TotalAmount   int                     `json:"total_amount"`
	TimeRange     wxCompleteTimeRange     `json:"time_range"`
	ProfitSharing bool                    `json:"profit_sharing"`
}

type wxCompletePostPayment struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

type wxCompleteTimeRange struct {
	EndTime string `json:"end_time"`
}

// wxCancelOrder 对齐 Java domain.wechat.WxCancelOrder（取消支付分订单请求）。
type wxCancelOrder struct {
	Appid     string `json:"appid"`
	ServiceID string `json:"service_id"`
	Reason    string `json:"reason"`
}

// ============================ 内部工具 ============================

// wxPayPost 使用微信支付 v3 客户端发送 POST 请求并把应答反序列化到 resp（resp 可为 nil）。
func wxPayPost(requestURL string, body, resp interface{}) error {
	client, err := getWxPayClient()
	if err != nil {
		return err
	}
	result, err := client.Post(context.Background(), requestURL, body)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	return core.UnMarshalResponse(result.Response, resp)
}

// wxPayGet 使用微信支付 v3 客户端发送 GET 请求并把应答反序列化到 resp。
func wxPayGet(requestURL string, resp interface{}) error {
	client, err := getWxPayClient()
	if err != nil {
		return err
	}
	result, err := client.Get(context.Background(), requestURL)
	if err != nil {
		return err
	}
	return core.UnMarshalResponse(result.Response, resp)
}

// hmacSHA256Hex 计算 HMAC-SHA256 并返回小写十六进制串（对齐 Java WxPayImp.hmacSHA256）。
func hmacSHA256Hex(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// getExtraData 拼装前端「小程序拉起支付分」的签名业务数据（对齐 Java WxPayImp.getExtraData）。
func getExtraData(mPackage string) openWeChatExtra {
	nonceStr := strings.ReplaceAll(uuid.NewString(), "-", "")
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signData := "mch_id=" + wxPayMerchantID +
		"&nonce_str=" + nonceStr +
		"&package=" + mPackage +
		"&sign_type=HMAC-SHA256" +
		"&timestamp=" + timestamp
	sign := strings.ToUpper(hmacSHA256Hex(signData+"&key="+wxPayAPIv3Key, wxPayAPIv3Key))
	return openWeChatExtra{
		MchID:     wxPayMerchantID,
		Package:   mPackage,
		Timestamp: timestamp,
		NonceStr:  nonceStr,
		SignType:  "HMAC-SHA256",
		Sign:      sign,
	}
}

// ============================ 对外方法 ============================

// createPayScoreOrder 创建微信支付分订单（对齐 WxPayService.createPayOrder）。
// 返回给前端的拉起支付参数；创建失败或状态非 CREATED 时 ExtraData 为 nil。
func createPayScoreOrder(orderID string, needProfitSharing bool, appID string) *OpenWeChat {
	if orderID == "" {
		return &OpenWeChat{}
	}
	if appID == "" {
		appID = wxPayAppID
	}

	req := payScoreRequest{
		OutOrderNo:          orderID,
		Appid:               appID,
		ServiceID:           wxPayServiceID,
		ServiceIntroduction: wxPayServiceName,
		PostPayments: []payScorePostPayment{
			{Name: "计费规则", Description: "电费最低0.23元/度起,服务费0.6元起,详情请见小程序", Count: 1},
		},
		TimeRange: payScoreTimeRange{StartTime: "OnAccept"},
		RiskFund:  payScoreRiskFund{Name: "DEPOSIT", Amount: 2},
		NotifyURL: wxPayScoreNotifyURL,
		// 需确认订单（对齐 Java need_user_confirm=true）
		NeedUserConfirm: true,
		ProfitSharing:   needProfitSharing,
	}

	var res wxCreateOrderRes
	if err := wxPayPost(wxPayBaseURL, req, &res); err != nil {
		logger.Mylog.Err(err).Msgf("创建微信支付分订单失败, orderId: %s", orderID)
		return &OpenWeChat{OutOrderNo: orderID}
	}
	if res.State != "CREATED" || res.Package == "" {
		logger.Mylog.Warn().Msgf("创建微信支付分订单返回状态异常, orderId: %s, state: %s", orderID, res.State)
		return &OpenWeChat{OutOrderNo: orderID}
	}

	extra := getExtraData(res.Package)
	return &OpenWeChat{
		BusinessType: "wxpayScoreUse",
		OutOrderNo:   orderID,
		ExtraData:    &extra,
	}
}

// payScoreAppID 根据订单消费类型返回支付分订单应使用的 appid。
// 踩踩停车（逸泊）用 yiparlAppID，其余用 wxPayAppID（对齐 Java OrderServiceImpl 创单时的 appId 选择）。
func payScoreAppID(orderID string) string {
	if order, err := repository.FindOrderByOrderid(orderID); err == nil && order != nil &&
		order.ConsumptionType == repository.ConsumeTypeCaiParking {
		return yiparlAppID
	}
	return wxPayAppID
}

// queryWxOrderState 查询微信支付分订单状态（对齐 WxPayService.onQueryWxOrder）。
// 返回 CREATED / DOING / DONE / REVOKED / EXPIRED；查询失败或单据不存在时返回 CANCEL
// （语义为「未使用微信支付分」，上层回退到余额支付）。
func queryWxOrderState(orderID string) string {
	if orderID == "" {
		return wxPayStateCancel
	}
	queryURL := wxPayBaseURL + "?" + url.Values{
		"out_order_no": {orderID},
		"service_id":   {wxPayServiceID},
		"appid":        {payScoreAppID(orderID)},
	}.Encode()

	var res wxQueryOrderRes
	if err := wxPayGet(queryURL, &res); err != nil {
		if strings.Contains(err.Error(), "ORDER_NOT_EXIST") {
			// 余额支付订单从未在微信支付分创单，查单返回「单据不存在」属正常情况，降级为 debug
			logger.Mylog.Debug().Msgf("查询微信支付分订单：单据不存在（余额支付订单属正常）, orderId: %s", orderID)
		} else {
			logger.Mylog.Warn().Msgf("查询微信支付分订单失败, orderId: %s, err: %v", orderID, err)
		}
		return wxPayStateCancel
	}
	if res.State == "" {
		return wxPayStateCancel
	}
	return res.State
}

// completeWxOrder 完结微信支付分订单（对齐 WxPayService.onCompleteWxOrder / onCompleteWxPrivateOrder /
// onCompleteWxTwiceOrder / onCompleteWxLockOrder，amount 单位：分）。
// 按订单消费类型构造「计费规则/费用明细」描述与分账标识。
func completeWxOrder(orderID string, amount int) {
	req := buildCompleteOrder(orderID, amount)
	if req == nil {
		return
	}
	completeURL := wxPayBaseURL + "/" + url.PathEscape(orderID) + "/complete"
	if err := wxPayPost(completeURL, req, nil); err != nil {
		logger.Mylog.Err(err).Msgf("完结微信支付分订单失败, orderId: %s, amount: %d", orderID, amount)
		return
	}
	logger.Mylog.Info().Msgf("完结微信支付分订单成功, orderId: %s, amount: %d", orderID, amount)
}

// buildCompleteOrder 按订单消费类型构造完结订单请求体（对齐 Java 各 onCompleteWx* 方法）。
func buildCompleteOrder(orderID string, amount int) *wxCompleteOrder {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		logger.Mylog.Warn().Msgf("completeWxOrder 查不到订单: %s", orderID)
		return nil
	}

	endTime := time.Now().Format("20060102150405")
	req := &wxCompleteOrder{
		Appid:       wxPayAppID,
		ServiceID:   wxPayServiceID,
		TotalAmount: amount,
		TimeRange:   wxCompleteTimeRange{EndTime: endTime},
	}

	switch order.ConsumptionType {
	case repository.ConsumeTypeYiXiang:
		// 驿享充电：分账（对齐 onCompleteWxOrder）
		totalFee, serviceFee := electronicFeesByOrderid(orderID)
		req.ProfitSharing = true
		req.PostPayments = []wxCompletePostPayment{
			{Name: "计费规则", Description: "电费最低0.23元/度起,服务费0.4元起,详情请见小程序", Count: 1},
			{Name: "费用明细", Description: "电费" + totalFee.String() + "元,服务费" + serviceFee.String() + "元", Count: 1},
		}
	case repository.ConsumeTypeNeighbor:
		// 邻享充电：不分账（对齐 onCompleteWxPrivateOrder）
		totalFee, serviceFee := repository.PrivateElectronicByOrderidAndPrivateUser(orderID)
		req.ProfitSharing = false
		req.PostPayments = []wxCompletePostPayment{
			{Name: "计费规则", Description: "电费最低0.23元/度起,服务费0.2元起,详情请见小程序", Count: 1},
			{Name: "费用明细", Description: "电费" + totalFee.String() + "元,服务费" + serviceFee.String() + "元", Count: 1},
		}
	case repository.ConsumeTypeTwice:
		// 二轮车充电：不分账（对齐 onCompleteWxTwiceOrder）
		totalFee := reOrderPowerNoServiceFees(orderID)
		serviceFee := decimal.NewFromInt(int64(amount)).Div(decimal.NewFromInt(100)).Sub(totalFee).Round(2)
		if serviceFee.IsNegative() {
			serviceFee = decimal.Zero
		}
		req.ProfitSharing = false
		req.PostPayments = []wxCompletePostPayment{
			{Name: "计费规则", Description: "电费最低0.23元/度起,服务费0.6元起,详情请见小程序", Count: 1},
			{Name: "费用明细", Description: "电费" + totalFee.String() + "元,服务费" + serviceFee.String() + "元", Count: 1},
		}
	default:
		// 踩踩停车：不分账，仅计费规则（对齐 onCompleteWxLockOrder）
		req.ProfitSharing = false
		req.PostPayments = []wxCompletePostPayment{
			{Name: "计费规则", Description: "停车费最低3元/半小时起,服务费0.6元起,详情请见小程序", Count: 1},
		}
	}
	return req
}

// cancelWxOrder 取消微信支付分订单（对齐 WxPayService.onCancelWxOrder）。
func cancelWxOrder(orderID string) {
	cancelURL := wxPayBaseURL + "/" + url.PathEscape(orderID) + "/cancel"
	req := wxCancelOrder{Appid: wxPayAppID, ServiceID: wxPayServiceID, Reason: "无需付款"}
	if err := wxPayPost(cancelURL, req, nil); err != nil {
		logger.Mylog.Err(err).Msgf("取消微信支付分订单失败, orderId: %s", orderID)
		return
	}
	logger.Mylog.Info().Msgf("取消微信支付分订单成功, orderId: %s", orderID)
}

// subtractWxDefaultAmount 完结订单并扣除违约金/定金（对齐 WxPayService.subtractDefaultAmount，amount 单位：分）。
func subtractWxDefaultAmount(orderID string, amount int) {
	completeURL := wxPayBaseURL + "/" + url.PathEscape(orderID) + "/complete"
	req := wxCompleteOrder{
		Appid:     wxPayAppID,
		ServiceID: wxPayServiceID,
		PostPayments: []wxCompletePostPayment{
			{Name: "费用明细", Description: "因当前时间距离预约时间较近,系统已自动按照扣费率扣除定金", Count: 1},
		},
		TotalAmount:   amount,
		TimeRange:     wxCompleteTimeRange{EndTime: time.Now().Format("20060102150405")},
		ProfitSharing: false,
	}
	if err := wxPayPost(completeURL, req, nil); err != nil {
		logger.Mylog.Err(err).Msgf("扣除定金失败, orderId: %s, amount: %d", orderID, amount)
		return
	}
	logger.Mylog.Info().Msgf("扣除定金成功, orderId: %s, amount: %d", orderID, amount)
}
