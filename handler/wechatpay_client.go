package handler

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"caicai-go/logger"
)

// ============================================================================
// 微信支付 v3 客户端（普通商户模式），基于官方 wechatpay-go SDK。
// 商户配置复用 handler/wxpay_controller.go 中的常量（商户号/证书序列号/APIv3 密钥），
// 商户私钥从 apiclient_key.pem 加载。
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
		privateKey, err := utils.LoadPrivateKeyWithPath("apiclient_key.pem")
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
// 返回微信退款状态：SUCCESS / PROCESSING / CHANGE / ABNORMAL / CLOSED。
func refundWxOrder(transactionID, outRefundNo string, refund, total int) (string, error) {
	client, err := getWxPayClient()
	if err != nil {
		return "", err
	}

	svc := refunddomestic.RefundsApiService{Client: client}
	resp, _, err := svc.Create(context.Background(), refunddomestic.CreateRequest{
		OutTradeNo:  core.String(transactionID),
		OutRefundNo: core.String(outRefundNo),
		Amount: &refunddomestic.AmountReq{
			Refund:   core.Int64(int64(refund)),
			Total:    core.Int64(int64(total)),
			Currency: core.String("CNY"),
		},
	})
	if err != nil {
		return "", err
	}
	if resp.Status == nil {
		return "", fmt.Errorf("微信退款返回缺少 status")
	}
	return string(*resp.Status), nil
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
