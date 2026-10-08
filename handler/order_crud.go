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
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
	"caicai-go/service"
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

// ============================ 金额查询 / 私桩费率 ============================

// queryFeeDecimal 执行返回单值金额的 SQL；结果为 NULL/无行时返回 nil（对齐 Java 返回 null）。
func queryFeeDecimal(sql string, args ...interface{}) *decimal.Decimal {
	var d decimal.Decimal
	if err := conf.Db.Raw(sql, args...).Row().Scan(&d); err != nil {
		return nil
	}
	return &d
}

// privateChargingFeeByOrderid 根据订单查询桩主设置的服务费率（order→private_place→private_charging）。
// 对齐 Java OrderElectronicServiceImpl.getElectronicByPrivateOrderidAndPrivateUser 中的取费逻辑。
func privateChargingFeeByOrderid(orderid string) *decimal.Decimal {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderid).First(&order).Error != nil {
		return nil
	}
	privatePlace := privatePlaceByID(order.SpacesID)
	if privatePlace == nil {
		return nil
	}
	chargingBean := privateChargingBySpacesNum(privatePlace.SpacesCode)
	if chargingBean == nil {
		return nil
	}
	fee := chargingBean.Fee
	return &fee
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

// GetValueByOrderid 对齐 Java getValueByOrderid：MAX(over_value)-MIN(start_value)，不除以 100。
func (c *OrderElectronicController) GetValueByOrderid(ctx *gin.Context) {
	d := queryFeeDecimal("SELECT (MAX(over_value)-MIN(start_value)) FROM tab_order_electronic WHERE order_id = ?", ctx.Query("orderid"))
	if d == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(*d))
}

// GetFeesByOrderid 对齐 Java getFeesByOrderid：电费 + 服务费合计（tab_price + tab_parking_spaces）。
func (c *OrderElectronicController) GetFeesByOrderid(ctx *gin.Context) {
	season := currentSeason()
	d := queryFeeDecimal(`SELECT SUM((e.over_value - e.start_value) / 100 * (
			CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END
			+
			CASE
				WHEN s.service_fee IS NOT NULL AND s.service_fee != 0 THEN s.service_fee
				WHEN s.service IS NOT NULL AND s.service != 0 THEN (s.service * 0.8)
				ELSE (p.service * 0.8)
			END
		))
		FROM tab_order_electronic e
		LEFT JOIN tab_price_time_`+season+` t ON t.id = e.price_time_id
		LEFT JOIN tab_price p ON p.id = t.price_id
		LEFT JOIN order_tbl o ON o.orderid = e.order_id
		LEFT JOIN tab_parking_spaces s ON s.id = o.spaces_id
		WHERE e.order_id = ?`, ctx.Query("orderid"))
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
	ctx.JSON(http.StatusOK, ResultSuccess(privateFeesByOrderid(ctx.Query("orderid"))))
}

// GetPrivateFeesByOrderidAndPrivateUser 对齐 Java getPrivateFeesByOrderidAndPrivateUser（fee 来自请求参数）。
func (c *OrderElectronicController) GetPrivateFeesByOrderidAndPrivateUser(ctx *gin.Context) {
	fee, err := decimal.NewFromString(ctx.Query("fee"))
	if err != nil {
		fee = decimal.Zero
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateFeesByOrderidAndPrivateUser(ctx.Query("orderid"), fee)))
}

// GetPrivateFeesByPrivateOrderid 对齐 Java getPrivateFeesByPrivateOrderid（桩主：仅 price.total）。
func (c *OrderElectronicController) GetPrivateFeesByPrivateOrderid(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(privateFeesByPrivateOrderid(ctx.Query("orderid"))))
}

// GetElectronicByPrivateOrderid 对齐 Java getElectronicByPrivateOrderid（private_charging_price_tbl.total/service）。
func (c *OrderElectronicController) GetElectronicByPrivateOrderid(ctx *gin.Context) {
	totalFee, serviceFee := privateElectronicByOrderid(ctx.Query("orderid"))
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{"totalFee": totalFee, "serviceFee": serviceFee}))
}

// GetElectronicByPrivateOrderidAndPrivateUser 对齐 Java getElectronicByPrivateOrderidAndPrivateUser
// （桩主设置的服务费率 fee 从私桩表查询，服务费 = fee*0.8）。
func (c *OrderElectronicController) GetElectronicByPrivateOrderidAndPrivateUser(ctx *gin.Context) {
	totalFee, serviceFee := privateElectronicByOrderidAndPrivateUser(ctx.Query("orderid"))
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{"totalFee": totalFee, "serviceFee": serviceFee}))
}

// GetPeriodFeesByOrderid 对齐 Java getPeriodFeesByOrderid（各时段电费/服务费明细）。
func (c *OrderElectronicController) GetPeriodFeesByOrderid(ctx *gin.Context) {
	season := currentSeason()
	rows, err := conf.Db.Raw(`SELECT
			e.create_time,
			e.update_time,
			CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END AS price_total,
			(e.over_value - e.start_value) / 100 * CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END AS total_fee,
			CASE
				WHEN s.service_fee IS NOT NULL AND s.service_fee != 0 THEN s.service_fee
				WHEN s.service IS NOT NULL AND s.service != 0 THEN (s.service * 0.8)
				ELSE (p.service * 0.8)
			END AS service_unit_price,
			(e.over_value - e.start_value) / 100 * CASE
				WHEN s.service_fee IS NOT NULL AND s.service_fee != 0 THEN s.service_fee
				WHEN s.service IS NOT NULL AND s.service != 0 THEN (s.service * 0.8)
				ELSE (p.service * 0.8)
			END AS service_fee
		FROM tab_order_electronic e
		LEFT JOIN tab_price_time_`+season+` t ON t.id = e.price_time_id
		LEFT JOIN tab_price p ON p.id = t.price_id
		LEFT JOIN order_tbl o ON o.orderid = e.order_id
		LEFT JOIN tab_parking_spaces s ON s.id = o.spaces_id
		WHERE e.order_id = ? AND e.over_value != e.start_value
		ORDER BY e.create_time`, ctx.Query("orderid")).Rows()
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess([]interface{}{}))
		return
	}
	defer rows.Close()

	list := make([]orderElectronicPeriodDTO, 0)
	for rows.Next() {
		var createTime, updateTime time.Time
		var priceTotal, totalFee, serviceUnitPrice, serviceFee sql.NullString
		if err := rows.Scan(&createTime, &updateTime, &priceTotal, &totalFee, &serviceUnitPrice, &serviceFee); err != nil {
			continue
		}
		vo := orderElectronicPeriodDTO{
			CreateTime:       timeToLocal(createTime),
			UpdateTime:       timeToLocal(updateTime),
			PriceTotal:       roundNullableDecimal(priceTotal, 4),
			TotalFee:         roundNullableDecimal(totalFee, 2),
			ServiceUnitPrice: roundNullableDecimal(serviceUnitPrice, 4),
			ServiceFee:       roundNullableDecimal(serviceFee, 2),
		}
		list = append(list, vo)
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
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
	ctx.JSON(http.StatusOK, ResultSuccess(privateFeesByPrivateOrderid(ctx.Query("orderId"))))
}

// GetOrderRechargeAmountByNeighbor 对齐 Java getOrderRechargeAmountByNeighbor（邻居：total + 桩主服务费）。
func (c *OrderPrivateController) GetOrderRechargeAmountByNeighbor(ctx *gin.Context) {
	orderID := ctx.Query("orderId")
	if orderID == "" || len(orderID) < 10 {
		ctx.JSON(http.StatusOK, ResultSuccess(decimal.Zero))
		return
	}
	fee := privateChargingFeeByOrderid(orderID)
	if fee == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(decimal.Zero))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateFeesByOrderidAndPrivateUser(orderID, *fee)))
}

// ============ OrderTwiceController（/orderTwice） ============

type OrderTwiceController struct{}

func (c *OrderTwiceController) GetByOrderid(ctx *gin.Context) {
	var o model.OrderTwiceTbl
	if err := conf.Db.Where("orderid = ?", ctx.Param("orderid")).First(&o).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(gin.H{}))
		return
	}
	// 对齐 Java ReOrderPowerMapper.getMaxPowerByOrderid：取 max_power 字段。
	var maxPower decimal.Decimal
	conf.Db.Raw("SELECT COALESCE(max_power,0) FROM re_order_electronic_tbl WHERE order_id = ? LIMIT 1", o.Orderid).Row().Scan(&maxPower)
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

// Statistics 对齐 Java RechargeOrderServiceImpl.getRechargeStatistics：
// 全部充值记录（不过滤状态）汇总 totalAmount/refundedAmount，remainingRefundAmount=用户当前余额。
func (c *RechargeOrderController) Statistics(ctx *gin.Context) {
	openid := ctx.Param("openid")

	var totalAmount, refundedAmount decimal.Decimal
	conf.Db.Model(&model.RechargeOrderTbl{}).Where("openid = ?", openid).
		Select("COALESCE(SUM(recharge_amount),0), COALESCE(SUM(refunded_amount),0)").Row().Scan(&totalAmount, &refundedAmount)

	var count int64
	conf.Db.Model(&model.RechargeOrderTbl{}).Where("openid = ?", openid).Count(&count)

	remainingRefundAmount := decimal.Zero
	var user model.UserTbl
	if conf.Db.Where("openid = ?", openid).First(&user).Error == nil {
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
	var r model.RechargeOrderTbl
	if err := conf.Db.Where("order_id = ?", req.OrderID).First(&r).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(404, "充值订单不存在"))
		return
	}
	res := conf.Db.Model(&model.RechargeOrderTbl{}).Where("order_id = ?", req.OrderID).Update("order_status", req.Status)
	if res.RowsAffected <= 0 {
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
		CreateTime:             time.Now(),
		UpdateTime:             time.Now(),
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&rec).Error == nil))
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
		ctx.JSON(http.StatusOK, ResultError(500, "提现记录不存在"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(withdrawalToDTO(w)))
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
	var w model.WithdrawalRecordTbl
	if err := conf.Db.Where("record_id = ?", req.RecordID).First(&w).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "提现记录不存在"))
		return
	}

	updates := map[string]interface{}{"withdrawal_status": req.Status, "update_time": time.Now()}
	if req.Status == withdrawalSuccess {
		updates["process_time"] = time.Now()
	} else if req.Status == withdrawalRejected {
		updates["reject_reason"] = req.RejectReason
	}
	res := conf.Db.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", req.RecordID).Updates(updates)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============================ 提现/退款业务逻辑 ============================

// applyRefund 对齐 Java WithdrawalRecordServiceImpl.applyRefund。
func (c *WithdrawalController) applyRefund(openid, rechargeOrderID string, refundAmount decimal.Decimal) (bool, string) {
	// 1. 验证退款金额
	if !refundAmount.IsPositive() {
		return false, "退款金额必须大于0"
	}

	// 2. 查询充值订单
	var rechargeOrder model.RechargeOrderTbl
	if err := conf.Db.Where("order_id = ?", rechargeOrderID).First(&rechargeOrder).Error; err != nil {
		return false, "充值订单不存在"
	}

	// 3. 验证订单状态和退款权限
	if rechargeOrder.OrderStatus != "已支付" {
		return false, "订单状态不支持退款"
	}

	// 4. 验证退款金额是否超过用户余额
	var user model.UserTbl
	if err := conf.Db.Where("openid = ?", openid).First(&user).Error; err != nil {
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
		CreateTime:             time.Now(),
		UpdateTime:             time.Now(),
	}
	if conf.Db.Omit("process_time").Create(&refundRecord).Error != nil {
		return false, "退款申请保存失败"
	}

	// 更新充值订单的退款申请次数
	conf.Db.Model(&model.RechargeOrderTbl{}).Where("order_id = ?", rechargeOrderID).
		Update("refund_apply_count", rechargeOrder.RefundApplyCount+1)

	return true, ""
}

// applyAndProcessRefund 对齐 Java WithdrawalRecordServiceImpl.applyAndProcessRefund。
func (c *WithdrawalController) applyAndProcessRefund(openid, rechargeOrderID string, refundAmount decimal.Decimal) (string, error) {
	ok, errMsg := c.applyRefund(openid, rechargeOrderID, refundAmount)
	if !ok {
		return "", fmt.Errorf("%s", errMsg)
	}

	// 查询刚创建的退款记录
	var latest model.WithdrawalRecordTbl
	if err := conf.Db.Where("openid = ? AND related_recharge_order_id = ? AND withdrawal_amount = ?",
		openid, rechargeOrderID, refundAmount).
		Order("create_time desc").First(&latest).Error; err != nil {
		return "", fmt.Errorf("未找到刚创建的退款记录")
	}

	return c.processRefund(latest.RecordID)
}

// processRefund 对齐 Java WithdrawalRecordServiceImpl.processRefund。
func (c *WithdrawalController) processRefund(recordID string) (string, error) {
	// 1. 验证退款记录
	var refundRecord model.WithdrawalRecordTbl
	if err := conf.Db.Where("record_id = ?", recordID).First(&refundRecord).Error; err != nil {
		return "", fmt.Errorf("退款记录不存在")
	}

	// 2. 验证充值订单
	var rechargeOrder model.RechargeOrderTbl
	if err := conf.Db.Where("order_id = ?", refundRecord.RelatedRechargeOrderID).First(&rechargeOrder).Error; err != nil {
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
		conf.Db.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", refundRecord.RecordID).
			Updates(map[string]interface{}{"withdrawal_status": withdrawalSuccess, "process_time": now})

		return c.processBalanceRefund(&refundRecord, &rechargeOrder)

	case "PROCESSING", "CHANGE":
		// 退款处理中（微信支付退款为异步）
		conf.Db.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", refundRecord.RecordID).
			Updates(map[string]interface{}{"withdrawal_status": withdrawalProcessing, "update_time": time.Now()})

		// 更新用户余额并记录余额明细（只要不是处理失败状态）
		if _, err := c.processBalanceRefund(&refundRecord, &rechargeOrder); err != nil {
			return "", err
		}
		return "退款申请已提交，正在处理中", nil

	default:
		// 其它状态视为失败，不更新余额
		conf.Db.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", refundRecord.RecordID).
			Updates(map[string]interface{}{
				"withdrawal_status": withdrawalRejected,
				"reject_reason":     "退款失败，状态：" + refundStatus,
				"process_time":      time.Now(),
			})
		return "", fmt.Errorf("退款处理失败：%s", refundStatus)
	}
}

// processBalanceRefund 对齐 Java WithdrawalRecordServiceImpl.processBalanceRefund。
func (c *WithdrawalController) processBalanceRefund(refundRecord *model.WithdrawalRecordTbl, rechargeOrder *model.RechargeOrderTbl) (string, error) {
	if err := c.doBalanceRefund(refundRecord, rechargeOrder); err != nil {
		// 退款失败，更新退款记录状态
		conf.Db.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", refundRecord.RecordID).
			Updates(map[string]interface{}{
				"withdrawal_status": withdrawalRejected,
				"reject_reason":     "余额退款失败：" + err.Error(),
				"process_time":      time.Now(),
			})
		return "", fmt.Errorf("余额退款失败：%s", err.Error())
	}
	return "余额退款成功", nil
}

// doBalanceRefund 余额充值订单退款的实际扣减逻辑。
// 多步写（扣余额 + 退款记录 + 充值订单 + 余额明细）包裹在事务内，扣款用原子 UPDATE，保证事务与金额安全。
func (c *WithdrawalController) doBalanceRefund(refundRecord *model.WithdrawalRecordTbl, rechargeOrder *model.RechargeOrderTbl) error {
	refundAmount := refundRecord.WithdrawalAmount
	amt := refundAmount.StringFixed(2)

	return conf.Db.Transaction(func(tx *gorm.DB) error {
		// 1. 扣减用户余额（原子 UPDATE，条件写进 WHERE 防止扣成负数）
		res := tx.Exec(
			"UPDATE user_tbl SET balans = balans - CAST(? AS DECIMAL(10,2)) WHERE openid = ? AND balans >= CAST(? AS DECIMAL(10,2))",
			amt, refundRecord.Openid, amt,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected <= 0 {
			return fmt.Errorf("退款金额超过用户当前余额")
		}

		// 2. 更新退款记录状态
		if err := tx.Model(&model.WithdrawalRecordTbl{}).Where("record_id = ?", refundRecord.RecordID).
			Updates(map[string]interface{}{"withdrawal_status": withdrawalSuccess, "process_time": time.Now()}).Error; err != nil {
			return err
		}

		// 3. 更新充值订单的退款信息
		rechargeOrder.RefundedAmount = rechargeOrder.RefundedAmount.Add(refundAmount)
		rechargeOrder.RemainingRefundAmount = rechargeOrder.RemainingRefundAmount.Sub(refundAmount)
		if err := tx.Model(&model.RechargeOrderTbl{}).Where("order_id = ?", rechargeOrder.OrderID).
			Updates(map[string]interface{}{
				"refunded_amount":         rechargeOrder.RefundedAmount,
				"remaining_refund_amount": rechargeOrder.RemainingRefundAmount,
			}).Error; err != nil {
			return err
		}

		// 4. 记录余额变动明细
		if err := tx.Create(&model.TabBalanceRecord{Openid: refundRecord.Openid, Type: "退费", Amount: refundAmount}).Error; err != nil {
			return err
		}

		return nil
	})
}
