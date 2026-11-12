package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"caicai-go/conf"
	"caicai-go/logger"
)

// ============================================================================
// 微信支付 v3 客户端（普通商户模式），基于官方 wechatpay-go SDK。
// 商户配置（商户号/证书序列号/APIv3 密钥/私钥路径等）从 conf.yaml 的 wxpay 段读取。
// ============================================================================

var (
	wxPayClient     *core.Client
	wxPayClientOnce sync.Once
	wxPayClientErr  error
)

// getWxPayClient 初始化微信支付 v3 客户端（平台证书自动下载，对齐 Java RSAAutoCertificateConfig），
// 具备「签名 / 验签 / 敏感字段加解密」能力，仅初始化一次。
func getWxPayClient() (*core.Client, error) {
	wxPayClientOnce.Do(func() {
		privateKey, err := utils.LoadPrivateKeyWithPath(conf.Cfg.Wxpay.PrivateKeyPath)
		if err != nil {
			wxPayClientErr = fmt.Errorf("加载商户私钥失败: %w", err)
			return
		}
		client, err := core.NewClient(context.Background(),
			option.WithWechatPayAutoAuthCipher(wxPayMerchantID, wxPayMchSerialNo, privateKey, wxPayAPIv3Key),
		)
		if err != nil {
			wxPayClientErr = fmt.Errorf("初始化微信支付客户端失败: %w", err)
			return
		}
		wxPayClient = client
	})
	if wxPayClientErr != nil {
		logger.Mylog.Error().Err(wxPayClientErr).Msg("微信支付客户端初始化失败")
	}
	return wxPayClient, wxPayClientErr
}

// prepayJsapi 微信支付 jsapi 统一下单，对齐 Java WeChatPayServiceApiV3.preOrder。
// totalFeeFen 金额单位分；返回 (prepayId, error)。
func prepayJsapi(totalFeeFen int, currency, openid, orderId string) (*jsapi.PrepayWithRequestPaymentResponse, error) {
	client, err := getWxPayClient()
	if err != nil {
		return nil, err
	}

	svc := jsapi.JsapiApiService{Client: client}
	resp, _, err := svc.PrepayWithRequestPayment(context.Background(), jsapi.PrepayRequest{
		Appid:       core.String(wxPayAppID),
		Mchid:       core.String(wxPayMerchantID),
		Description: core.String("余额充值"),
		OutTradeNo:  core.String(orderId),
		NotifyUrl:   core.String(wxPayNotifyURL),
		Amount: &jsapi.Amount{
			Total:    core.Int64(int64(totalFeeFen)),
			Currency: core.String(currency),
		},
		Payer: &jsapi.Payer{Openid: core.String(openid)},
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// refundWxOrder 申请退款（对齐 Java WxPayService.refunds），金额单位分。
// 第一个参数为微信支付订单号（transaction_id），与 Java 一致发送 transaction_id 字段。
// 返回微信退款状态：SUCCESS / PROCESSING / CHANGE / ABNORMAL / CLOSED。
func refundWxOrder(transactionID, outRefundNo string, refund, total int) (string, error) {
	res, err := createWxRefund(transactionID, outRefundNo, refund, total, false)
	if err != nil {
		return "", err
	}
	return res.Status, nil
}

// refundWxOrderByOutTradeNo 按商户订单号（out_trade_no）申请退款。
// 用于旧数据「未保存微信支付订单号」时回退用系统订单号退款的场景（字段正确，优于 Java 的兼容分支）。
func refundWxOrderByOutTradeNo(outTradeNo, outRefundNo string, refund, total int) (string, error) {
	res, err := createWxRefund(outTradeNo, outRefundNo, refund, total, true)
	if err != nil {
		return "", err
	}
	return res.Status, nil
}

// refundWxOrderFull 申请退款并返回完整退款响应（对齐 Java WxPayService.refunds 返回 RefundResult）。
// 供 /wx/refunds 接口使用，返回字段与 Java model.wechat.RefundResult 一致。
func refundWxOrderFull(transactionID, outRefundNo string, refund, total int) (*refundResult, error) {
	return createWxRefund(transactionID, outRefundNo, refund, total, false)
}

// createWxRefund 校验参数并调用微信退款接口，返回完整退款响应。
// useOutTradeNo=true 时按商户订单号退款，否则按微信支付订单号退款。
func createWxRefund(idField, outRefundNo string, refund, total int, useOutTradeNo bool) (*refundResult, error) {
	// 参数校验（对齐 Java WxPayImp.refunds）
	if strings.TrimSpace(idField) == "" {
		return nil, fmt.Errorf("微信支付订单号不能为空")
	}
	if strings.TrimSpace(outRefundNo) == "" {
		return nil, fmt.Errorf("商户退款单号不能为空")
	}
	if refund <= 0 {
		return nil, fmt.Errorf("退款金额必须大于0")
	}
	if total <= 0 {
		return nil, fmt.Errorf("原订单金额必须大于0")
	}
	if refund > total {
		return nil, fmt.Errorf("退款金额不能超过原订单金额")
	}

	client, err := getWxPayClient()
	if err != nil {
		return nil, err
	}

	req := refunddomestic.CreateRequest{
		OutRefundNo: core.String(outRefundNo),
		NotifyUrl:   core.String(wxPayRefundNotifyURL),
		Amount: &refunddomestic.AmountReq{
			Refund:   core.Int64(int64(refund)),
			Total:    core.Int64(int64(total)),
			Currency: core.String("CNY"),
		},
	}
	if useOutTradeNo {
		req.OutTradeNo = core.String(idField)
	} else {
		req.TransactionId = core.String(idField)
	}

	svc := refunddomestic.RefundsApiService{Client: client}
	resp, _, err := svc.Create(context.Background(), req)
	if err != nil {
		return nil, err
	}
	if resp.Status == nil {
		return nil, fmt.Errorf("微信退款返回缺少 status")
	}
	return refundResultFromSDK(resp), nil
}

// refundResultFromSDK 将微信退款 SDK 响应转换为对齐 Java RefundResult 的结构。
func refundResultFromSDK(resp *refunddomestic.Refund) *refundResult {
	if resp == nil {
		return &refundResult{}
	}
	r := &refundResult{
		RefundID:            derefStr(resp.RefundId),
		TransactionID:       derefStr(resp.TransactionId),
		OutTradeNo:          derefStr(resp.OutTradeNo),
		OutRefundNo:         derefStr(resp.OutRefundNo),
		Channel:             derefStr(resp.Channel),
		UserReceivedAccount: derefStr(resp.UserReceivedAccount),
		SuccessTime:         timeToLocal(resp.SuccessTime),
		CreateTime:          timeToLocal(resp.CreateTime),
		Status:              derefStr(resp.Status),
		FundsAccount:        derefStr(resp.FundsAccount),
	}
	if resp.Amount != nil {
		r.Amount = &refundAmountDTO{
			Total:            int(derefInt64(resp.Amount.Total)),
			Refund:           int(derefInt64(resp.Amount.Refund)),
			PayerTotal:       int(derefInt64(resp.Amount.PayerTotal)),
			PayerRefund:      int(derefInt64(resp.Amount.PayerRefund)),
			SettlementRefund: int(derefInt64(resp.Amount.SettlementRefund)),
			SettlementTotal:  int(derefInt64(resp.Amount.SettlementTotal)),
			DiscountRefund:   int(derefInt64(resp.Amount.DiscountRefund)),
			Currency:         derefStr(resp.Amount.Currency),
		}
	}
	return r
}

// derefStr 解引用 string 别名指针（*string / *Channel / *Status / *FundsAccount），nil 返回空串。
func derefStr[T ~string](v *T) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

// derefInt64 解引用 *int64，nil 返回 0。
func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// refundQueryWxOrder 查询退款状态（按商户退款单号），返回状态字符串，对齐 Java WxPayService.refundQuery。
func refundQueryWxOrder(outRefundNo string) (string, error) {
	res, err := refundQueryWxOrderFull(outRefundNo)
	if err != nil {
		return "", err
	}
	return res.Status, nil
}

// refundQueryWxOrderFull 查询退款并返回完整结果（对齐 Java domain.RefundQueryResult，驼峰字段名）。
func refundQueryWxOrderFull(outRefundNo string) (*refundQueryResult, error) {
	client, err := getWxPayClient()
	if err != nil {
		return nil, err
	}
	svc := refunddomestic.RefundsApiService{Client: client}
	resp, _, err := svc.QueryByOutRefundNo(context.Background(), refunddomestic.QueryByOutRefundNoRequest{
		OutRefundNo: core.String(outRefundNo),
	})
	if err != nil {
		return nil, err
	}
	if resp.Status == nil {
		return nil, fmt.Errorf("微信退款查询返回缺少 status")
	}
	return refundQueryResultFromSDK(resp), nil
}

// refundQueryResultFromSDK 将微信退款查询 SDK 响应转换为对齐 Java RefundQueryResult 的结构（驼峰字段名）。
func refundQueryResultFromSDK(resp *refunddomestic.Refund) *refundQueryResult {
	if resp == nil {
		return &refundQueryResult{}
	}
	r := &refundQueryResult{
		RefundID:            derefStr(resp.RefundId),
		OutRefundNo:         derefStr(resp.OutRefundNo),
		Transaction:         derefStr(resp.TransactionId),
		Channel:             derefStr(resp.Channel),
		UserReceivedAccount: derefStr(resp.UserReceivedAccount),
		Status:              derefStr(resp.Status),
	}
	if resp.SuccessTime != nil {
		r.SuccessTime = resp.SuccessTime.Format(time.RFC3339)
	}
	if resp.CreateTime != nil {
		r.CreateTime = resp.CreateTime.Format(time.RFC3339)
	}
	if resp.Amount != nil {
		r.Amount = &refundQueryAmountDTO{
			Total:            int(derefInt64(resp.Amount.Total)),
			Refund:           int(derefInt64(resp.Amount.Refund)),
			PayerTotal:       int(derefInt64(resp.Amount.PayerTotal)),
			PayerRefund:      int(derefInt64(resp.Amount.PayerRefund)),
			SettlementTotal:  int(derefInt64(resp.Amount.SettlementTotal)),
			SettlementRefund: int(derefInt64(resp.Amount.SettlementRefund)),
			DiscountRefund:   int(derefInt64(resp.Amount.DiscountRefund)),
			Currency:         derefStr(resp.Amount.Currency),
		}
	}
	return r
}

// parseWxPayNotify 验签并解密微信支付回调通知，解密后的 resource 反序列化到 content。
// content 通常传 *payments.Transaction（JSAPI 支付成功通知）；返回通知请求（含 EventType）。
func parseWxPayNotify(r *http.Request, content interface{}) (*notify.Request, error) {
	visitor := downloader.MgrInstance().GetCertificateVisitor(wxPayMerchantID)
	handler, err := notify.NewRSANotifyHandler(wxPayAPIv3Key, verifiers.NewSHA256WithRSAVerifier(visitor))
	if err != nil {
		return nil, err
	}
	return handler.ParseNotifyRequest(r.Context(), r, content)
}
