package handler

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/repository"
	"caicai-go/service"
)

// ============ DTO ============

type tabOrderElectronicDTO struct {
	CreateTime  *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime  *LocalDateTime `json:"updateTime,omitempty"`
	OrderID     string         `json:"orderId,omitempty"`
	StartValue  float64        `json:"startValue,omitempty"`
	OverValue   float64        `json:"overValue,omitempty"`
	ID          int32          `json:"id"`
	PriceTimeID int32          `json:"priceTimeId,omitempty"`
}

func tabOrderElectronicToDTO(e model.TabOrderElectronic) tabOrderElectronicDTO {
	return tabOrderElectronicDTO{
		ID:          e.ID,
		OrderID:     e.OrderID,
		StartValue:  e.StartValue.InexactFloat64(),
		OverValue:   e.OverValue.InexactFloat64(),
		PriceTimeID: e.PriceTimeID,
		CreateTime:  timeToLocal(e.CreateTime),
		UpdateTime:  timeToLocal(e.UpdateTime),
	}
}

type orderPrivateDTO struct {
	Orderid       string         `json:"orderid"`
	IsPrivateUser int32          `json:"isPrivateUser,omitempty"`
	CreateTime    *LocalDateTime `json:"createTime,omitempty"`
}

func orderPrivateToDTO(o model.OrderPrivateTbl) orderPrivateDTO {
	return orderPrivateDTO{Orderid: o.Orderid, IsPrivateUser: o.IsPrivateUser, CreateTime: timeToLocal(o.CreateTime)}
}

type orderTwiceDTO struct {
	Orderid     string          `json:"orderid"`
	PowerRate   decimal.Decimal `json:"powerRate,omitempty"`
	ServiceRate decimal.Decimal `json:"serviceRate,omitempty"`
	CreateTime  *LocalDateTime  `json:"createTime,omitempty"`
	UpdateTime  *LocalDateTime  `json:"updateTime,omitempty"`
}

func orderTwiceToDTO(o model.OrderTwiceTbl) orderTwiceDTO {
	return orderTwiceDTO{Orderid: o.Orderid, PowerRate: o.PowerRate, ServiceRate: o.ServiceRate, CreateTime: timeToLocal(o.CreateTime), UpdateTime: timeToLocal(o.UpdateTime)}
}

type rechargeOrderDTO struct {
	OrderID               string          `json:"orderId"`
	Openid                string          `json:"openid,omitempty"`
	RechargeAmount        decimal.Decimal `json:"rechargeAmount,omitempty"`
	RefundedAmount        decimal.Decimal `json:"refundedAmount,omitempty"`
	RemainingRefundAmount decimal.Decimal `json:"remainingRefundAmount,omitempty"`
	OrderStatus           string          `json:"orderStatus,omitempty"`
	CreateTime            *LocalDateTime  `json:"createTime,omitempty"`
	PaymentTime           *LocalDateTime  `json:"paymentTime,omitempty"`
	RefundTime            *LocalDateTime  `json:"refundTime,omitempty"`
	WechatTransactionID   string          `json:"wechatTransactionId,omitempty"`
}

func rechargeOrderToDTO(r model.RechargeOrderTbl) rechargeOrderDTO {
	return rechargeOrderDTO{
		OrderID: r.OrderID, Openid: r.Openid, RechargeAmount: r.RechargeAmount,
		RefundedAmount: r.RefundedAmount, RemainingRefundAmount: r.RemainingRefundAmount,
		OrderStatus: r.OrderStatus, CreateTime: timeToLocal(r.CreateTime),
		PaymentTime: timeToLocal(r.PaymentTime), RefundTime: timeToLocal(r.RefundTime),
		WechatTransactionID: r.WechatTransactionID,
	}
}

type withdrawalDTO struct {
	RecordID               string          `json:"recordId"`
	Openid                 string          `json:"openid,omitempty"`
	WithdrawalAmount       decimal.Decimal `json:"withdrawalAmount,omitempty"`
	RefundOrderID          string          `json:"refundOrderId,omitempty"`
	RelatedRechargeOrderID string          `json:"relatedRechargeOrderId,omitempty"`
	WithdrawalStatus       string          `json:"withdrawalStatus,omitempty"`
	CreateTime             *LocalDateTime  `json:"createTime,omitempty"`
	ProcessTime            *LocalDateTime  `json:"processTime,omitempty"`
	RejectReason           string          `json:"rejectReason,omitempty"`
	WithdrawalType         string          `json:"withdrawalType,omitempty"`
	WxRefundID             string          `json:"wxRefundId,omitempty"`
}

func withdrawalToDTO(w model.WithdrawalRecordTbl) withdrawalDTO {
	return withdrawalDTO{
		RecordID: w.RecordID, Openid: w.Openid, WithdrawalAmount: w.WithdrawalAmount,
		RefundOrderID: w.RefundOrderID, RelatedRechargeOrderID: w.RelatedRechargeOrderID,
		WithdrawalStatus: w.WithdrawalStatus, CreateTime: timeToLocal(w.CreateTime),
		ProcessTime: timeToLocal(w.ProcessTime), RejectReason: w.RejectReason,
		WithdrawalType: w.WithdrawalType, WxRefundID: w.WxRefundID,
	}
}

// orderElectronicPeriodDTO 对应 Java OrderElectronicPeriodVo（各时段电费/服务费明细）。
type orderElectronicPeriodDTO struct {
	CreateTime       *LocalDateTime   `json:"createTime,omitempty"`
	UpdateTime       *LocalDateTime   `json:"updateTime,omitempty"`
	PriceTotal       *decimal.Decimal `json:"priceTotal,omitempty"`
	TotalFee         *decimal.Decimal `json:"totalFee,omitempty"`
	ServiceUnitPrice *decimal.Decimal `json:"serviceUnitPrice,omitempty"`
	ServiceFee       *decimal.Decimal `json:"serviceFee,omitempty"`
}

// electronicFeesByOrderid 计算订单的电费与服务费（对齐 Java getElectronicByOrderid，季节电价表）。
func electronicFeesByOrderid(orderid string) (totalFee, serviceFee decimal.Decimal) {
	return repository.ElectronicFeesByOrderid(orderid)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// roundNullableDecimal 将可空字符串金额解析并按指定小数位舍入（Java setScale HALF_UP）。
func roundNullableDecimal(ns sql.NullString, scale int32) *decimal.Decimal {
	if !ns.Valid {
		return nil
	}
	d, err := decimal.NewFromString(ns.String)
	if err != nil {
		return nil
	}
	v := d.Round(scale)
	return &v
}

// ============================ 提现状态常量（对齐 Java RefundStatusEnum） ============================

const (
	withdrawalApplying   = "退款申请中" // APPLYING
	withdrawalProcessing = "处理中"   // PROCESSING
	withdrawalSuccess    = "已处理"   // SUCCESS
	withdrawalRejected   = "已拒绝"   // REJECTED
)

// simpleUUID 对齐 Java hutool IdUtil.simpleUUID()（32 位无横线小写十六进制）。
func simpleUUID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// ============ OrderElectronicController（/orderElectronic） ============

type OrderElectronicController struct{}

func (c *OrderElectronicController) GetById(ctx *gin.Context) {
	e, err := repository.TabOrderElectronicByID(ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(tabOrderElectronicToDTO(*e)))
}

func (c *OrderElectronicController) Add(ctx *gin.Context) {
	var e model.TabOrderElectronic
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.InsertTabOrderElectronic(&e) == nil))
}

func (c *OrderElectronicController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	total, rows := repository.PageTabOrderElectronics(pageOffset(current, size), size)
	list := make([]tabOrderElectronicDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, tabOrderElectronicToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetValueByOrderid 对齐 Java getValueByOrderid：MAX(over_value)-MIN(start_value)，不除以 100。
func (c *OrderElectronicController) GetValueByOrderid(ctx *gin.Context) {
	d := repository.TabOrderElectronicValueByOrderID(ctx.Query("orderid"))
	if d == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(*d))
}

// GetFeesByOrderid 对齐 Java getFeesByOrderid：电费 + 服务费合计（tab_price + tab_parking_spaces）。
func (c *OrderElectronicController) GetFeesByOrderid(ctx *gin.Context) {
	d := repository.TabOrderElectronicFeesByOrderID(ctx.Query("orderid"))
	if d == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(d.Round(2)))
}

func (c *OrderElectronicController) GetElectronicByOrderid(ctx *gin.Context) {
	totalFee, serviceFee := electronicFeesByOrderid(ctx.Query("orderid"))
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{"totalFee": totalFee, "serviceFee": serviceFee}))
}

// GetPrivateFeesByOrderid 对齐 Java getPrivateFeesByOrderid（private_charging_price_tbl.total + service）。
func (c *OrderElectronicController) GetPrivateFeesByOrderid(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(repository.PrivateFeesByOrderid(ctx.Query("orderid"))))
}

// GetPrivateFeesByOrderidAndPrivateUser 对齐 Java getPrivateFeesByOrderidAndPrivateUser（fee 来自请求参数）。
func (c *OrderElectronicController) GetPrivateFeesByOrderidAndPrivateUser(ctx *gin.Context) {
	fee, err := decimal.NewFromString(ctx.Query("fee"))
	if err != nil {
		fee = decimal.Zero
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.PrivateFeesByOrderidAndPrivateUser(ctx.Query("orderid"), fee)))
}

// GetPrivateFeesByPrivateOrderid 对齐 Java getPrivateFeesByPrivateOrderid（桩主：仅 price.total）。
func (c *OrderElectronicController) GetPrivateFeesByPrivateOrderid(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(repository.PrivateFeesByPrivateOrderid(ctx.Query("orderid"))))
}

// GetElectronicByPrivateOrderid 对齐 Java getElectronicByPrivateOrderid（private_charging_price_tbl.total/service）。
func (c *OrderElectronicController) GetElectronicByPrivateOrderid(ctx *gin.Context) {
	totalFee, serviceFee := repository.PrivateElectronicByOrderid(ctx.Query("orderid"))
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{"totalFee": totalFee, "serviceFee": serviceFee}))
}

// GetElectronicByPrivateOrderidAndPrivateUser 对齐 Java getElectronicByPrivateOrderidAndPrivateUser
// （桩主设置的服务费率 fee 从私桩表查询，服务费 = fee*0.8）。
func (c *OrderElectronicController) GetElectronicByPrivateOrderidAndPrivateUser(ctx *gin.Context) {
	totalFee, serviceFee := repository.PrivateElectronicByOrderidAndPrivateUser(ctx.Query("orderid"))
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{"totalFee": totalFee, "serviceFee": serviceFee}))
}

// GetPeriodFeesByOrderid 对齐 Java getPeriodFeesByOrderid（各时段电费/服务费明细）。
func (c *OrderElectronicController) GetPeriodFeesByOrderid(ctx *gin.Context) {
	rows, err := repository.TabOrderElectronicPeriodsByOrderID(ctx.Query("orderid"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess([]interface{}{}))
		return
	}

	list := make([]orderElectronicPeriodDTO, 0)
	for _, r := range rows {
		vo := orderElectronicPeriodDTO{
			CreateTime:       timeToLocal(timePtr(r.CreateTime)),
			UpdateTime:       timeToLocal(timePtr(r.UpdateTime)),
			PriceTotal:       roundNullableDecimal(r.PriceTotal, 4),
			TotalFee:         roundNullableDecimal(r.TotalFee, 2),
			ServiceUnitPrice: roundNullableDecimal(r.ServiceUnitPrice, 4),
			ServiceFee:       roundNullableDecimal(r.ServiceFee, 2),
		}
		list = append(list, vo)
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// ============ OrderPrivateController（/orderPrivate） ============

type OrderPrivateController struct{}

func (c *OrderPrivateController) GetById(ctx *gin.Context) {
	o := repository.OrderPrivateByOrderid(ctx.Param("orderid"))
	if o == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(orderPrivateToDTO(*o)))
}

func (c *OrderPrivateController) Add(ctx *gin.Context) {
	var o model.OrderPrivateTbl
	if err := ctx.ShouldBindJSON(&o); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.InsertOrderPrivate(&o) == nil))
}

// GetOrderState 对齐 Java OrderPrivateServiceImpl.getOrderState（Redis 键 PrivateOrderState:{orderId}）。
func (c *OrderPrivateController) GetOrderState(ctx *gin.Context) {
	state := service.GetRedisValue("PrivateOrderState:" + ctx.Query("orderId"))
	if state == "" {
		state = "暂时无法获取到订单信息,请稍后重试"
	}
	ctx.JSON(http.StatusOK, ResultSuccess(state))
}

// GetOrderRechargeAmountByOwner 对齐 Java getOrderRechargeAmountByOwner（桩主：getPrivateFeesByPrivateOrderid）。
func (c *OrderPrivateController) GetOrderRechargeAmountByOwner(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(repository.PrivateFeesByPrivateOrderid(ctx.Query("orderId"))))
}

// GetOrderRechargeAmountByNeighbor 对齐 Java getOrderRechargeAmountByNeighbor（邻居：total + 桩主服务费）。
func (c *OrderPrivateController) GetOrderRechargeAmountByNeighbor(ctx *gin.Context) {
	orderID := ctx.Query("orderId")
	if orderID == "" || len(orderID) < 10 {
		ctx.JSON(http.StatusOK, ResultSuccess(decimal.Zero))
		return
	}
	fee := repository.PrivateChargingFeeByOrderid(orderID)
	if fee == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(decimal.Zero))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.PrivateFeesByOrderidAndPrivateUser(orderID, *fee)))
}

// ============ OrderTwiceController（/orderTwice） ============

type OrderTwiceController struct{}

func (c *OrderTwiceController) GetByOrderid(ctx *gin.Context) {
	o, err := repository.OrderTwiceByOrderid(ctx.Param("orderid"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(gin.H{}))
		return
	}
	// 对齐 Java ReOrderPowerMapper.getMaxPowerByOrderid：取 max_power 字段。
	maxPower := repository.ReOrderPowerMaxByOrderid(o.Orderid)
	logger.Mylog.Info().Msgf("====>> %#v <<=====", maxPower.Round(2))
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{
		"totalFee":   o.PowerRate.Round(2),
		"serviceFee": o.ServiceRate.Round(2),
		"maxPower":   maxPower.Round(2),
	}))
}

func (c *OrderTwiceController) Save(ctx *gin.Context) {
	var o model.OrderTwiceTbl
	if err := ctx.ShouldBindJSON(&o); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.InsertOrderTwice(&o) == nil))
}

// ============ RechargeOrderController（/recharge） ============

type RechargeOrderController struct{}

func (c *RechargeOrderController) List(ctx *gin.Context) {
	rows := repository.RechargeOrdersByOpenidStatus(ctx.Param("openid"), "已支付")
	list := make([]rechargeOrderDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, rechargeOrderToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *RechargeOrderController) Detail(ctx *gin.Context) {
	r, err := repository.RechargeOrderByOrderID(ctx.Param("orderId"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(404, "充值订单不存在"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(rechargeOrderToDTO(*r)))
}

// Statistics 对齐 Java RechargeOrderServiceImpl.getRechargeStatistics：
// 全部充值记录（不过滤状态）汇总 totalAmount/refundedAmount，remainingRefundAmount=用户当前余额。
func (c *RechargeOrderController) Statistics(ctx *gin.Context) {
	openid := ctx.Param("openid")

	totalAmount, refundedAmount := repository.RechargeOrderSumsByOpenid(openid)
	count := repository.CountRechargeOrdersByOpenid(openid)

	remainingRefundAmount := decimal.Zero
	if user := repository.FindUserByOpenid(openid); user != nil {
		remainingRefundAmount = user.Balans
	}

	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{
		"totalAmount":           totalAmount.Round(2),
		"refundedAmount":        refundedAmount.Round(2),
		"remainingRefundAmount": remainingRefundAmount.Round(2),
		"rechargeCount":         count,
	}))
}

func (c *RechargeOrderController) UpdateStatus(ctx *gin.Context) {
	var req struct {
		OrderID string `json:"orderId"`
		Status  string `json:"status"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "更新充值订单状态失败"))
		return
	}
	if _, err := repository.RechargeOrderByOrderID(req.OrderID); err != nil {
		ctx.JSON(http.StatusOK, ResultError(404, "充值订单不存在"))
		return
	}
	if repository.UpdateRechargeOrderStatus(req.OrderID, req.Status) <= 0 {
		ctx.JSON(http.StatusOK, ResultError(500, "充值订单状态更新失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ WithdrawalController（/withdrawal） ============

type WithdrawalController struct{}

// Apply 对齐 Java applyWithdrawal：仅创建提现记录（状态=退款申请中）。
func (c *WithdrawalController) Apply(ctx *gin.Context) {
	var req struct {
		Openid                 string          `json:"openid"`
		Amount                 decimal.Decimal `json:"amount"`
		RelatedRechargeOrderID string          `json:"relatedRechargeOrderId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	rec := model.WithdrawalRecordTbl{
		RecordID:               simpleUUID(),
		Openid:                 req.Openid,
		WithdrawalAmount:       req.Amount,
		RefundOrderID:          "",
		RelatedRechargeOrderID: req.RelatedRechargeOrderID,
		WithdrawalStatus:       withdrawalApplying,
		WithdrawalType:         "提现",
		CreateTime:             timePtr(time.Now()),
		UpdateTime:             timePtr(time.Now()),
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.InsertWithdrawalRecord(&rec) == nil))
}

// Refund 对齐 Java applyRefund：校验金额/订单状态/用户余额，创建退款记录并累计退款申请次数。
func (c *WithdrawalController) Refund(ctx *gin.Context) {
	var req struct {
		Openid          string          `json:"openid"`
		RechargeOrderID string          `json:"rechargeOrderId"`
		RefundAmount    decimal.Decimal `json:"refundAmount"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if ok, errMsg := c.applyRefund(req.Openid, req.RechargeOrderID, req.RefundAmount); !ok {
		ctx.JSON(http.StatusOK, ResultError(500, errMsg))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// RefundAndProcess 对齐 Java applyAndProcessRefund：申请退款后立即处理。
func (c *WithdrawalController) RefundAndProcess(ctx *gin.Context) {
	var req struct {
		Openid          string          `json:"openid"`
		RechargeOrderID string          `json:"rechargeOrderId"`
		RefundAmount    decimal.Decimal `json:"refundAmount"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	result, err := c.applyAndProcessRefund(req.Openid, req.RechargeOrderID, req.RefundAmount)
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, err.Error()))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(result))
}

// ProcessRefund 对齐 Java processRefund：调用微信退款并按状态更新记录/充值订单/余额。
func (c *WithdrawalController) ProcessRefund(ctx *gin.Context) {
	var req struct {
		RecordID string `json:"recordId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	result, err := c.processRefund(req.RecordID)
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, err.Error()))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(result))
}

func (c *WithdrawalController) List(ctx *gin.Context) {
	rows := repository.WithdrawalRecordsByOpenid(ctx.Param("openid"))
	list := make([]withdrawalDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, withdrawalToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *WithdrawalController) Detail(ctx *gin.Context) {
	w, err := repository.WithdrawalRecordByRecordID(ctx.Param("recordId"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "提现记录不存在"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(withdrawalToDTO(*w)))
}

// UpdateStatus 对齐 Java updateWithdrawalStatus：状态为「已处理」写处理时间，「已拒绝」写拒绝原因。
func (c *WithdrawalController) UpdateStatus(ctx *gin.Context) {
	var req struct {
		RecordID     string `json:"recordId"`
		Status       string `json:"status"`
		RejectReason string `json:"rejectReason"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if _, err := repository.WithdrawalRecordByRecordID(req.RecordID); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "提现记录不存在"))
		return
	}

	updates := map[string]interface{}{"withdrawal_status": req.Status, "update_time": time.Now()}
	if req.Status == withdrawalSuccess {
		updates["process_time"] = time.Now()
	} else if req.Status == withdrawalRejected {
		updates["reject_reason"] = req.RejectReason
	}
	res := repository.UpdateWithdrawalRecord(req.RecordID, updates)
	ctx.JSON(http.StatusOK, ResultSuccess(res > 0))
}

// ============================ 提现/退款业务逻辑 ============================

// applyRefund 对齐 Java WithdrawalRecordServiceImpl.applyRefund。
func (c *WithdrawalController) applyRefund(openid, rechargeOrderID string, refundAmount decimal.Decimal) (bool, string) {
	// 1. 验证退款金额
	if !refundAmount.IsPositive() {
		return false, "退款金额必须大于0"
	}

	// 2. 查询充值订单
	rechargeOrder, err := repository.RechargeOrderByOrderID(rechargeOrderID)
	if err != nil {
		return false, "充值订单不存在"
	}

	// 3. 验证订单状态和退款权限
	if rechargeOrder.OrderStatus != "已支付" {
		return false, "订单状态不支持退款"
	}

	// 4. 验证退款金额是否超过用户余额
	user := repository.FindUserByOpenid(openid)
	if user == nil {
		return false, "用户不存在"
	}
	if refundAmount.GreaterThan(user.Balans) {
		return false, "退款金额超过用户余额"
	}

	// 5. 创建退款记录
	refundOrderID := "REFUND_" + strconv.FormatInt(time.Now().UnixMilli(), 10) + "_" + simpleUUID()[:8]
	refundRecord := model.WithdrawalRecordTbl{
		RecordID:               simpleUUID(),
		Openid:                 openid,
		WithdrawalAmount:       refundAmount,
		RefundOrderID:          refundOrderID,
		RelatedRechargeOrderID: rechargeOrderID,
		WithdrawalStatus:       withdrawalApplying,
		WithdrawalType:         "退款",
		CreateTime:             timePtr(time.Now()),
		UpdateTime:             timePtr(time.Now()),
	}
	if err := repository.InsertWithdrawalRecord(&refundRecord); err != nil {
		return false, "退款申请保存失败"
	}

	// 更新充值订单的退款申请次数
	repository.UpdateRefundApplyCount(rechargeOrderID, rechargeOrder.RefundApplyCount+1)

	return true, ""
}

// applyAndProcessRefund 对齐 Java WithdrawalRecordServiceImpl.applyAndProcessRefund。
func (c *WithdrawalController) applyAndProcessRefund(openid, rechargeOrderID string, refundAmount decimal.Decimal) (string, error) {
	ok, errMsg := c.applyRefund(openid, rechargeOrderID, refundAmount)
	if !ok {
		return "", fmt.Errorf("%s", errMsg)
	}

	// 查询刚创建的退款记录
	latest, err := repository.LatestWithdrawalRecord(openid, rechargeOrderID, refundAmount)
	if err != nil {
		return "", fmt.Errorf("未找到刚创建的退款记录")
	}

	return c.processRefund(latest.RecordID)
}

// processRefund 对齐 Java WithdrawalRecordServiceImpl.processRefund。
func (c *WithdrawalController) processRefund(recordID string) (string, error) {
	// 1. 验证退款记录
	refundRecord, err := repository.WithdrawalRecordByRecordID(recordID)
	if err != nil {
		return "", fmt.Errorf("退款记录不存在")
	}

	// 2. 验证充值订单
	rechargeOrder, err := repository.RechargeOrderByOrderID(refundRecord.RelatedRechargeOrderID)
	if err != nil {
		return "", fmt.Errorf("关联的充值订单不存在")
	}

	// 3. 获取微信支付交易号（为空则回退系统订单号，兼容旧数据）
	transactionID := rechargeOrder.WechatTransactionID
	if strings.TrimSpace(transactionID) == "" {
		transactionID = refundRecord.RelatedRechargeOrderID
	}

	// 调用微信支付退款接口。金额单位：分。
	refundFen := refundRecord.WithdrawalAmount.Mul(decimal.NewFromInt(100)).IntPart()
	totalFen := rechargeOrder.RechargeAmount.Mul(decimal.NewFromInt(100)).IntPart()
	refundStatus, err := refundWxOrder(transactionID, refundRecord.RefundOrderID, int(refundFen), int(totalFen))
	if err != nil {
		return "", fmt.Errorf("微信退款失败: %w", err)
	}

	switch refundStatus {
	case "SUCCESS":
		// 退款成功处理：充值订单退款信息与用户余额统一由 processBalanceRefund 处理，
		// 避免 refundedAmount/remainingRefundAmount 被重复累加（Java 原逻辑存在重复累加 bug，此处修正）。
		now := time.Now()
		repository.UpdateWithdrawalRecord(refundRecord.RecordID, map[string]interface{}{"withdrawal_status": withdrawalSuccess, "process_time": now})

		return c.processBalanceRefund(refundRecord, rechargeOrder)

	case "PROCESSING", "CHANGE":
		// 退款处理中（微信支付退款为异步）
		repository.UpdateWithdrawalRecord(refundRecord.RecordID, map[string]interface{}{"withdrawal_status": withdrawalProcessing, "update_time": time.Now()})

		// 更新用户余额并记录余额明细（只要不是处理失败状态）
		if _, err := c.processBalanceRefund(refundRecord, rechargeOrder); err != nil {
			return "", err
		}
		return "退款申请已提交，正在处理中", nil

	default:
		// 其它状态视为失败，不更新余额
		repository.UpdateWithdrawalRecord(refundRecord.RecordID, map[string]interface{}{
			"withdrawal_status": withdrawalRejected,
			"reject_reason":     "退款失败，状态：" + refundStatus,
			"process_time":      time.Now(),
		})
		return "", fmt.Errorf("退款处理失败：%s", refundStatus)
	}
}

// processBalanceRefund 对齐 Java WithdrawalRecordServiceImpl.processBalanceRefund。
func (c *WithdrawalController) processBalanceRefund(refundRecord *model.WithdrawalRecordTbl, rechargeOrder *model.RechargeOrderTbl) (string, error) {
	if err := repository.BalanceRefundTx(refundRecord, rechargeOrder); err != nil {
		// 退款失败，更新退款记录状态
		repository.UpdateWithdrawalRecord(refundRecord.RecordID, map[string]interface{}{
			"withdrawal_status": withdrawalRejected,
			"reject_reason":     "余额退款失败：" + err.Error(),
			"process_time":      time.Now(),
		})
		return "", fmt.Errorf("余额退款失败：%s", err.Error())
	}
	return "余额退款成功", nil
}
