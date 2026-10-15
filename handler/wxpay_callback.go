package handler

import (
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/repository"
)

// ============================================================================
// 微信支付分回调处理 + 分账触发，对齐 Java WxPayImp.handlePaymentSuccess /
// OrderServiceImpl.triggerProfitSharing / executeProfitSharingAfterPayment。
// ============================================================================

// extractTransactionID 从查询/通知结果的 collection.details 中提取微信交易号。
func (res *wxQueryOrderRes) extractTransactionID() string {
	if res == nil || res.Collection == nil {
		return ""
	}
	for _, d := range res.Collection.Details {
		if d.TransactionID != "" {
			return d.TransactionID
		}
	}
	return ""
}

// handlePayscorePaid 处理支付分「支付成功」事件：保存微信交易号并触发分账。
func handlePayscorePaid(content *wxQueryOrderRes) {
	if content == nil || content.OutOrderNo == "" {
		logger.Mylog.Error().Msg("支付成功回调中商户订单号为空，无法处理")
		return
	}
	transactionID := content.extractTransactionID()
	if transactionID != "" {
		saveOrderTransactionID(content.OutOrderNo, transactionID)
		return
	}
	logger.Mylog.Warn().Msgf("支付分回调中未包含微信交易号，主动查询服务订单状态 - 订单号: %s", content.OutOrderNo)
	queryAndSaveTransactionID(content.OutOrderNo)
}

// saveOrderTransactionID 把微信交易号写回订单，成功后异步触发分账（对齐 Java saveTransactionId）。
func saveOrderTransactionID(outOrderNo, transactionID string) {
	rowsAffected, err := repository.UpdateOrderWechatTransactionID(outOrderNo, transactionID)
	if err != nil {
		logger.Mylog.Error().Err(err).Msgf("更新订单微信交易号失败 - 订单号: %s", outOrderNo)
		return
	}
	if rowsAffected == 0 {
		logger.Mylog.Warn().Msgf("更新订单微信交易号无匹配订单 - 订单号: %s", outOrderNo)
		return
	}
	logger.Mylog.Info().Msgf("成功更新订单微信交易号 - 订单号: %s, 微信交易号: %s", outOrderNo, transactionID)
	go triggerProfitSharing(outOrderNo)
}

// queryAndSaveTransactionID 查询服务订单状态并保存交易号（对齐 Java queryAndSaveTransactionId）。
func queryAndSaveTransactionID(outOrderNo string) {
	if outOrderNo == "" {
		return
	}
	// 等待微信支付分订单状态更新后再查询
	time.Sleep(2 * time.Second)
	res := queryWxOrderDetail(outOrderNo)
	if res != nil {
		if id := res.extractTransactionID(); id != "" {
			saveOrderTransactionID(outOrderNo, id)
			return
		}
	}
	// 第一次未取到交易号，再等待后重试
	time.Sleep(3 * time.Second)
	if res = queryWxOrderDetail(outOrderNo); res != nil {
		if id := res.extractTransactionID(); id != "" {
			saveOrderTransactionID(outOrderNo, id)
			return
		}
	}
	logger.Mylog.Warn().Msgf("重试查询仍未获取到交易号 - 订单号: %s", outOrderNo)
}

// queryWxOrderDetail 查询支付分订单详情（含 collection 收款信息）。
func queryWxOrderDetail(outOrderNo string) *wxQueryOrderRes {
	queryURL := wxPayBaseURL + "?" + url.Values{
		"out_order_no": {outOrderNo},
		"service_id":   {wxPayServiceID},
		"appid":        {wxPayAppID},
	}.Encode()
	res := new(wxQueryOrderRes)
	if err := wxPayGet(queryURL, res); err != nil {
		logger.Mylog.Warn().Msgf("查询服务订单状态失败 - 订单号: %s, err: %v", outOrderNo, err)
		return nil
	}
	return res
}

// triggerProfitSharing 触发分账（对齐 Java OrderServiceImpl.triggerProfitSharing）。
// 仅对驿享充电订单执行；未配置分账时走解冻；按模式计算分账金额后调用分账接口。
func triggerProfitSharing(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		logger.Mylog.Warn().Msgf("触发分账失败：未找到订单 - 订单号: %s", orderID)
		return
	}
	if order.ConsumptionType != repository.ConsumeTypeYiXiang {
		logger.Mylog.Info().Msgf("订单类型不符合分账条件，跳过分账 - 订单号: %s, 类型: %s", orderID, order.ConsumptionType)
		return
	}

	config, err := repository.ProfitSharingConfigBySpacesID(order.SpacesID)
	if err != nil {
		logger.Mylog.Info().Msgf("该车位未配置分账或已禁用，调用解冻接口释放资金 - 订单号: %s", orderID)
		unfreezeOrderInternal(orderID)
		return
	}

	mode := config.Mode
	if mode == 0 {
		mode = 1
	}
	totalFee, serviceFee := electronicFeesByOrderid(orderID)
	basicConsumption := order.BasicConsumption

	var amount decimal.Decimal
	if mode == 1 {
		amount = serviceFee // 模式1：仅服务费
	} else {
		amount = basicConsumption.Sub(totalFee) // 模式2：服务费 + 超时占位费
	}
	if amount.IsNegative() {
		amount = decimal.Zero
	}
	if amount.IsZero() {
		logger.Mylog.Info().Msgf("分账金额为0，跳过分账 - 订单号: %s, 模式: %d", orderID, mode)
		return
	}

	executeProfitSharing(orderID, amount, *config)
}

// 微信分账 / 解冻接口 URL（对齐 Java ProfitSharingServiceImpl 的常量）。
const (
	profitSharingURL         = "https://api.mch.weixin.qq.com/v3/profitsharing/orders"
	profitSharingUnfreezeURL = "https://api.mch.weixin.qq.com/v3/profitsharing/orders/unfreeze"
)

// profitSharingRequest 请求分账请求体（对齐 Java buildProfitSharingRequest）。
type profitSharingRequest struct {
	Appid           string                  `json:"appid"`
	TransactionID   string                  `json:"transaction_id"`
	OutOrderNo      string                  `json:"out_order_no"`
	Receivers       []profitSharingReceiver `json:"receivers"`
	UnfreezeUnsplit bool                    `json:"unfreeze_unsplit"`
}

// profitSharingReceiver 分账接收方（对齐 Java buildProfitSharingRequest 的 receiver）。
type profitSharingReceiver struct {
	Type        string `json:"type"`
	Account     string `json:"account"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

// profitSharingUnfreezeRequest 解冻剩余资金请求体（对齐 Java unfreezeOrder）。
type profitSharingUnfreezeRequest struct {
	Appid         string `json:"appid"`
	TransactionID string `json:"transaction_id"`
	OutOrderNo    string `json:"out_order_no"`
	Description   string `json:"description"`
}

// profitSharingResponse 分账/解冻响应（取 state 判断结果）。
type profitSharingResponse struct {
	State         string `json:"state"`
	OrderID       string `json:"order_id"`
	OutOrderNo    string `json:"out_order_no"`
	TransactionID string `json:"transaction_id"`
}

// executeProfitSharing 执行微信分账（对齐 Java ProfitSharingService.executeProfitSharing）。
// amount 为分账总额（元），按 config.SharingRatio 计算接收方实际分账金额。
func executeProfitSharing(orderID string, amount decimal.Decimal, config model.TabProfitSharingConfig) {
	if strings.TrimSpace(config.ReceiverMchID) == "" {
		logger.Mylog.Error().Msgf("分账接收方商户号未配置 - 车位号: %d", config.SpacesID)
		return
	}
	amountFen := amount.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
	// 分账金额 = 总额(分) × 比例(%) / 100，向下取整（对齐 Java calculateSharingAmount）
	sharingAmount := amountFen * int64(config.SharingRatio) / 100
	if sharingAmount <= 0 {
		logger.Mylog.Warn().Msgf("分账金额为0，跳过分账 - 订单号: %s", orderID)
		return
	}

	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || strings.TrimSpace(order.WechatTransactionID) == "" {
		logger.Mylog.Warn().Msgf("未找到微信交易号，跳过分账 - 订单号: %s", orderID)
		return
	}

	desc := config.Description
	if desc == "" {
		desc = "平台服务费分账"
	}
	req := profitSharingRequest{
		Appid:         wxPayAppID,
		TransactionID: order.WechatTransactionID,
		OutOrderNo:    "PS_" + orderID,
		Receivers: []profitSharingReceiver{
			{Type: "MERCHANT_ID", Account: config.ReceiverMchID, Amount: sharingAmount, Description: desc},
		},
		UnfreezeUnsplit: true,
	}
	var resp profitSharingResponse
	if err := wxPayPost(profitSharingURL, req, &resp); err != nil {
		logger.Mylog.Err(err).Msgf("分账请求失败 - 订单号: %s", orderID)
		return
	}
	if resp.State == "PROCESSING" || resp.State == "FINISHED" {
		logger.Mylog.Info().Msgf("分账请求成功 - 订单号: %s, 金额: %d分, 接收方: %s, 状态: %s",
			orderID, sharingAmount, config.ReceiverName, resp.State)
		return
	}
	logger.Mylog.Error().Msgf("分账失败 - 订单号: %s, 状态: %s", orderID, resp.State)
}

// unfreezeOrderInternal 解冻订单资金（对齐 Java ProfitSharingService.unfreezeOrder）。
func unfreezeOrderInternal(orderID string) {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || strings.TrimSpace(order.WechatTransactionID) == "" {
		logger.Mylog.Warn().Msgf("解冻失败：订单不存在或无微信交易号 - 订单号: %s", orderID)
		return
	}

	req := profitSharingUnfreezeRequest{
		Appid:         wxPayAppID,
		TransactionID: order.WechatTransactionID,
		OutOrderNo:    "UF_" + orderID,
		Description:   "分账配置已禁用，解冻剩余资金",
	}
	// 解冻接口成功可能返回空 body 或 JSON，这里仅以 HTTP 成功为准（对齐 Java 的空响应即成功）。
	if err := wxPayPost(profitSharingUnfreezeURL, req, nil); err != nil {
		logger.Mylog.Err(err).Msgf("解冻资金失败 - 订单号: %s", orderID)
		return
	}
	logger.Mylog.Info().Msgf("解冻资金成功 - 订单号: %s, 交易号: %s", orderID, order.WechatTransactionID)
}
