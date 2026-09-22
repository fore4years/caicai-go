package handler

import (
	"github.com/shopspring/decimal"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ DTO ============

type tabOrderElectronicDTO struct {
	ID          int32          `json:"id"`
	OrderID     string         `json:"orderId,omitempty"`
	StartValue  float64        `json:"startValue,omitempty"`
	OverValue   float64        `json:"overValue,omitempty"`
	PriceTimeID int32          `json:"priceTimeId,omitempty"`
	CreateTime  *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime  *LocalDateTime `json:"updateTime,omitempty"`
}

func tabOrderElectronicToDTO(e model.TabOrderElectronic) tabOrderElectronicDTO {
	return tabOrderElectronicDTO{
		ID:          e.ID,
		OrderID:     e.OrderID,
		StartValue:  e.StartValue,
		OverValue:   e.OverValue,
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

// electronicFeesByOrderid 计算订单的电费与服务费（对齐 Java getElectronicByOrderid，季节电价表）。
func electronicFeesByOrderid(orderid string) (totalFee, serviceFee decimal.Decimal) {
	season := currentSeason()
	sql := `SELECT
		COALESCE(SUM((e.over_value - e.start_value) / 100 * CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END),0),
		COALESCE(SUM((e.over_value - e.start_value) / 100 * CASE WHEN s.service_fee IS NOT NULL AND s.service_fee != 0 THEN s.service_fee WHEN s.service IS NOT NULL AND s.service != 0 THEN (s.service*0.8) ELSE (p.service*0.8) END),0)
		FROM tab_order_electronic e
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = e.price_time_id
		LEFT JOIN tab_price p ON p.id = t.price_id
		LEFT JOIN order_tbl o ON o.orderid = e.order_id
		LEFT JOIN tab_parking_spaces s ON s.id = o.spaces_id
		WHERE e.order_id = ?`
	var tf, sf decimal.Decimal
	conf.Db.Raw(sql, orderid).Row().Scan(&tf, &sf)
	return tf.Round(2), sf.Round(2)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ============ OrderElectronicController（/orderElectronic） ============

type OrderElectronicController struct{}

func (c *OrderElectronicController) GetById(ctx *gin.Context) {
	var e model.TabOrderElectronic
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&e).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(tabOrderElectronicToDTO(e)))
}

func (c *OrderElectronicController) Add(ctx *gin.Context) {
	var e model.TabOrderElectronic
	if err := ctx.ShouldBindJSON(&e); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&e).Error == nil))
}

func (c *OrderElectronicController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.TabOrderElectronic{}).Count(&total)
	var rows []model.TabOrderElectronic
	conf.Db.Model(&model.TabOrderElectronic{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]tabOrderElectronicDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, tabOrderElectronicToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (c *OrderElectronicController) GetValueByOrderid(ctx *gin.Context) {
	var v float64
	conf.Db.Raw("SELECT COALESCE(SUM(over_value - start_value),0)/100 FROM tab_order_electronic WHERE order_id = ?", ctx.Query("orderid")).Row().Scan(&v)
	ctx.JSON(http.StatusOK, ResultSuccess(round2(v)))
}

func (c *OrderElectronicController) GetFeesByOrderid(ctx *gin.Context) {
	totalFee, _ := electronicFeesByOrderid(ctx.Query("orderid"))
	ctx.JSON(http.StatusOK, ResultSuccess(totalFee))
}

func (c *OrderElectronicController) GetElectronicByOrderid(ctx *gin.Context) {
	totalFee, serviceFee := electronicFeesByOrderid(ctx.Query("orderid"))
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{"totalFee": totalFee, "serviceFee": serviceFee}))
}

func (c *OrderElectronicController) GetPrivateFeesByOrderid(ctx *gin.Context) {
	c.GetFeesByOrderid(ctx)
}

func (c *OrderElectronicController) GetPrivateFeesByOrderidAndPrivateUser(ctx *gin.Context) {
	c.GetFeesByOrderid(ctx)
}

func (c *OrderElectronicController) GetPrivateFeesByPrivateOrderid(ctx *gin.Context) {
	c.GetFeesByOrderid(ctx)
}

func (c *OrderElectronicController) GetElectronicByPrivateOrderid(ctx *gin.Context) {
	c.GetElectronicByOrderid(ctx)
}

func (c *OrderElectronicController) GetElectronicByPrivateOrderidAndPrivateUser(ctx *gin.Context) {
	c.GetElectronicByOrderid(ctx)
}

func (c *OrderElectronicController) GetPeriodFeesByOrderid(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess([]interface{}{}))
}

// ============ OrderPrivateController（/orderPrivate） ============

type OrderPrivateController struct{}

func (c *OrderPrivateController) GetById(ctx *gin.Context) {
	var o model.OrderPrivateTbl
	if err := conf.Db.Where("orderid = ?", ctx.Param("orderid")).First(&o).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(orderPrivateToDTO(o)))
}

func (c *OrderPrivateController) Add(ctx *gin.Context) {
	var o model.OrderPrivateTbl
	if err := ctx.ShouldBindJSON(&o); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&o).Error == nil))
}

func (c *OrderPrivateController) GetOrderState(ctx *gin.Context) {
	var state string
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", ctx.Query("orderId")).Pluck("state", &state)
	ctx.JSON(http.StatusOK, ResultSuccess(state))
}

func (c *OrderPrivateController) GetOrderRechargeAmountByOwner(ctx *gin.Context) {
	totalFee, _ := electronicFeesByOrderid(ctx.Query("orderId"))
	ctx.JSON(http.StatusOK, ResultSuccess(totalFee))
}

func (c *OrderPrivateController) GetOrderRechargeAmountByNeighbor(ctx *gin.Context) {
	totalFee, _ := electronicFeesByOrderid(ctx.Query("orderId"))
	ctx.JSON(http.StatusOK, ResultSuccess(totalFee))
}

// ============ OrderTwiceController（/orderTwice） ============

type OrderTwiceController struct{}

func (c *OrderTwiceController) GetByOrderid(ctx *gin.Context) {
	var o model.OrderTwiceTbl
	if err := conf.Db.Where("orderid = ?", ctx.Param("orderid")).First(&o).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(gin.H{"totalFee": 0, "serviceFee": 0, "maxPower": 0}))
		return
	}
	var maxPower float64
	conf.Db.Raw("SELECT COALESCE(MAX(power),0) FROM re_order_electronic_tbl WHERE order_id = ?", o.Orderid).Row().Scan(&maxPower)
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{
		"totalFee":   o.PowerRate.Round(2),
		"serviceFee": o.ServiceRate.Round(2),
		"maxPower":   round2(maxPower),
	}))
}

func (c *OrderTwiceController) Save(ctx *gin.Context) {
	var o model.OrderTwiceTbl
	if err := ctx.ShouldBindJSON(&o); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&o).Error == nil))
}

// ============ RechargeOrderController（/recharge） ============

type RechargeOrderController struct{}

func (c *RechargeOrderController) List(ctx *gin.Context) {
	var rows []model.RechargeOrderTbl
	conf.Db.Where("openid = ? AND order_status = ?", ctx.Param("openid"), "已支付").Order("create_time desc").Find(&rows)
	list := make([]rechargeOrderDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, rechargeOrderToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *RechargeOrderController) Detail(ctx *gin.Context) {
	var r model.RechargeOrderTbl
	if err := conf.Db.Where("order_id = ?", ctx.Param("orderId")).First(&r).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(404, "充值订单不存在"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(rechargeOrderToDTO(r)))
}

func (c *RechargeOrderController) Statistics(ctx *gin.Context) {
	var total decimal.Decimal
	var count int64
	conf.Db.Model(&model.RechargeOrderTbl{}).
		Where("openid = ? AND order_status = ?", ctx.Param("openid"), "已支付").
		Select("COALESCE(SUM(recharge_amount),0), COUNT(*)").Row().Scan(&total, &count)
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{"totalAmount": total.Round(2), "count": count}))
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
	res := conf.Db.Model(&model.RechargeOrderTbl{}).Where("order_id = ?", req.OrderID).Update("order_status", req.Status)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ WithdrawalController（/withdrawal） ============

type WithdrawalController struct{}

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
		RecordID:               "WD" + time.Now().Format("20060102150405"),
		Openid:                 req.Openid,
		WithdrawalAmount:       req.Amount,
		RelatedRechargeOrderID: req.RelatedRechargeOrderID,
		WithdrawalStatus:       "申请中",
		WithdrawalType:         "提现",
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&rec).Error == nil))
}

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
	rec := model.WithdrawalRecordTbl{
		RecordID:               "RF" + time.Now().Format("20060102150405"),
		Openid:                 req.Openid,
		WithdrawalAmount:       req.RefundAmount,
		RelatedRechargeOrderID: req.RechargeOrderID,
		WithdrawalStatus:       "申请中",
		WithdrawalType:         "退款",
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&rec).Error == nil))
}

func (c *WithdrawalController) RefundAndProcess(ctx *gin.Context) {
	c.Refund(ctx)
}

func (c *WithdrawalController) ProcessRefund(ctx *gin.Context) {
	var req struct {
		RecordID string `json:"recordId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", req.RecordID).
		Updates(map[string]interface{}{"withdrawal_status": "已处理", "process_time": time.Now()})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *WithdrawalController) List(ctx *gin.Context) {
	var rows []model.WithdrawalRecordTbl
	conf.Db.Where("openid = ?", ctx.Param("openid")).Order("create_time desc").Find(&rows)
	list := make([]withdrawalDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, withdrawalToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *WithdrawalController) Detail(ctx *gin.Context) {
	var w model.WithdrawalRecordTbl
	if err := conf.Db.Where("record_id = ?", ctx.Param("recordId")).First(&w).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(withdrawalToDTO(w)))
}

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
	res := conf.Db.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", req.RecordID).
		Updates(map[string]interface{}{"withdrawal_status": req.Status, "reject_reason": req.RejectReason})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}
