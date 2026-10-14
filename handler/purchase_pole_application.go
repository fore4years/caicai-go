package handler

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ PurchasePoleApplicationController（/purchasePoleApplication） ============

type PurchasePoleApplicationController struct{}

// UpdateOrderStatus 对应 Java PurchasePoleApplicationController.updateOrderStatus。
func (c *PurchasePoleApplicationController) UpdateOrderStatus(ctx *gin.Context) {
	res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
		Where("order_number = ?", ctx.Query("orderNumber")).
		Update("order_status", ctx.Query("orderStatus"))
	if res.RowsAffected == 0 {
		ctx.JSON(http.StatusOK, ResultError(400, "订单状态更新失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// GetByOrderStatus 对应 Java PurchasePoleApplicationController.getByOrderStatus（按申请时间倒序）。
func (c *PurchasePoleApplicationController) GetByOrderStatus(ctx *gin.Context) {
	var rows []model.PurchasePoleApplicationTbl
	conf.Db.Where("order_status = ?", ctx.Param("orderStatus")).Order("apply_time desc").Find(&rows)
	list := make([]purchasePoleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, purchasePoleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// Create 对应 Java PurchasePoleApplicationController.createPurchasePoleApplication。
func (c *PurchasePoleApplicationController) Create(ctx *gin.Context) {
	var req struct {
		Openid               string          `json:"openid"`
		Name                 string          `json:"name"`
		IDNumber             string          `json:"idNumber"`
		SelectedPackage      string          `json:"selectedPackage"`
		SelectedColor        string          `json:"selectedColor"`
		InstallationDistance float64         `json:"installationDistance"`
		UsageScenario        string          `json:"usageScenario"`
		InstallationAddress  string          `json:"installationAddress"`
		DetailedAddress      string          `json:"detailedAddress"`
		PhoneNumber          string          `json:"phoneNumber"`
		ActualPayment        decimal.Decimal `json:"actualPayment"`
		InstallationDate     string          `json:"installationDate"`
		PaymentType          string          `json:"paymentType"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "购桩申请创建异常: "+err.Error()))
		return
	}

	// 实付款为 0 → 待发货，否则待支付。
	orderStatus := "待支付"
	if req.ActualPayment.IsZero() {
		orderStatus = "待发货"
	}
	usageScenario := req.UsageScenario
	if strings.TrimSpace(usageScenario) == "" {
		usageScenario = "未指定"
	}

	p := model.PurchasePoleApplicationTbl{
		OrderNumber:          generateOrderNumber(),
		Openid:               req.Openid,
		Name:                 req.Name,
		IDNumber:             req.IDNumber,
		SelectedPackage:      req.SelectedPackage,
		SelectedColor:        req.SelectedColor,
		InstallationDistance: decimal.NewFromFloat(req.InstallationDistance),
		UsageScenario:        usageScenario,
		InstallationAddress:  req.InstallationAddress,
		DetailedAddress:      req.DetailedAddress,
		PhoneNumber:          req.PhoneNumber,
		ActualPayment:        req.ActualPayment,
		InstallationDate:     req.InstallationDate,
		PaymentType:          req.PaymentType,
		OrderStatus:          orderStatus,
		ApplyTime:            timePtr(time.Now()),
	}
	if err := conf.Db.Create(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "购桩申请创建失败"))
		return
	}

	// 返回最新创建的订单号（按申请时间倒序第一条）。
	var latest model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("openid = ?", req.Openid).Order("apply_time desc").First(&latest).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "订单创建成功但未找到订单信息"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(latest.OrderNumber))
}

// generateOrderNumber 生成订单号：PPA + 年月日 + 6 位随机数（对齐 Java generateOrderNumber）。
func generateOrderNumber() string {
	return fmt.Sprintf("PPA%s%d", time.Now().Format("20060102"), rand.IntN(900000)+100000)
}

// GetCurrentUserApplication 对应 Java getCurrentUserApplication：取当前登录用户。
func (c *PurchasePoleApplicationController) GetCurrentUserApplication(ctx *gin.Context) {
	openid := ctx.GetString("openid")
	var rows []model.PurchasePoleApplicationTbl
	conf.Db.Where("openid = ?", openid).Order("apply_time desc").Find(&rows)
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultError(500, "未找到购桩申请信息"))
		return
	}
	list := make([]purchasePoleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, purchasePoleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetByOpenid 对应 Java getByOpenid（路径参数 openid，按申请时间倒序）。
func (c *PurchasePoleApplicationController) GetByOpenid(ctx *gin.Context) {
	openid := ctx.Param("openid")
	if openid == "" {
		openid = ctx.GetString("openid")
	}
	var rows []model.PurchasePoleApplicationTbl
	conf.Db.Where("openid = ?", openid).Order("apply_time desc").Find(&rows)
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultError(500, "未找到购桩申请信息"))
		return
	}
	list := make([]purchasePoleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, purchasePoleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// DeleteByOrderNumber 对应 Java deleteByOrderNumber。
func (c *PurchasePoleApplicationController) DeleteByOrderNumber(ctx *gin.Context) {
	res := conf.Db.Where("order_number = ?", ctx.Param("orderNumber")).Delete(&model.PurchasePoleApplicationTbl{})
	if res.RowsAffected == 0 {
		ctx.JSON(http.StatusOK, ResultError(400, "购桩申请删除失败，订单可能不存在"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// Pay 对应 Java pay：校验订单存在且为「待支付」，调用微信支付创建订单。
func (c *PurchasePoleApplicationController) Pay(ctx *gin.Context) {
	orderNumber := ctx.Param("orderNumber")
	var p model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", orderNumber).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "订单不存在"))
		return
	}
	if p.OrderStatus != "待支付" {
		ctx.JSON(http.StatusOK, ResultError(400, "订单状态不正确，无法支付"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(createPayScoreOrder(orderNumber, false, appid)))
}

// UpdatePaymentInfo 对应 Java updatePaymentInfo：更新交易号、支付时间并置状态为已支付。
func (c *PurchasePoleApplicationController) UpdatePaymentInfo(ctx *gin.Context) {
	payTime := time.Now()
	if v := ctx.Query("payTime"); v != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.Local); err == nil {
			payTime = t
		}
	}
	res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
		Where("order_number = ?", ctx.Query("orderNumber")).
		Updates(map[string]interface{}{
			"transaction_id": ctx.Query("transactionId"),
			"pay_time":       payTime,
			"order_status":   "已支付",
		})
	if res.RowsAffected == 0 {
		ctx.JSON(http.StatusOK, ResultError(400, "支付信息更新失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// GetByOrderNumber 对应 Java getByOrderNumber。
func (c *PurchasePoleApplicationController) GetByOrderNumber(ctx *gin.Context) {
	var p model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", ctx.Param("orderNumber")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "未找到购桩申请信息"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(purchasePoleToDTO(p)))
}

// ApplyAndProcessRefund 对应 Java applyAndProcessRefund：一次性完成申请与退款。
func (c *PurchasePoleApplicationController) ApplyAndProcessRefund(ctx *gin.Context) {
	var req struct {
		OrderNumber  string `json:"orderNumber"`
		RefundReason string `json:"refundReason"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: "+err.Error()))
		return
	}

	// 1. 验证订单是否存在
	var app model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", req.OrderNumber).First(&app).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 购桩订单不存在"))
		return
	}

	// 2. 验证订单状态是否支持退款
	if app.OrderStatus != "已支付" {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 订单状态不支持退款，当前状态："+app.OrderStatus))
		return
	}

	// 3. 验证支付信息是否完整
	if strings.TrimSpace(app.TransactionID) == "" {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 订单支付信息不完整，无法进行退款"))
		return
	}

	// 4. 生成退款单号
	refundOrderID := "REFUND_PPA_" + fmt.Sprintf("%d", time.Now().UnixMilli())

	// 5. 金额（元 → 分）
	refundAmount := int(app.ActualPayment.Mul(decimal.NewFromInt(100)).IntPart())
	totalAmount := refundAmount
	refundStatus, err := refundWxOrder(app.TransactionID, refundOrderID, refundAmount, totalAmount)
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 微信退款失败: "+err.Error()))
		return
	}

	// 6. 处理退款结果
	switch refundStatus {
	case "SUCCESS":
		conf.Db.Model(&model.PurchasePoleApplicationTbl{}).Where("order_number = ?", req.OrderNumber).Updates(map[string]interface{}{
			"order_status":  "已退款",
			"refund_reason": req.RefundReason,
			"refund_time":   time.Now(),
		})
		ctx.JSON(http.StatusOK, ResultSuccess("退款成功"))
	case "PROCESSING", "CHANGE":
		conf.Db.Model(&model.PurchasePoleApplicationTbl{}).Where("order_number = ?", req.OrderNumber).Updates(map[string]interface{}{
			"order_status":  "退款中",
			"refund_reason": req.RefundReason,
			"refund_time":   time.Now(),
		})
		ctx.JSON(http.StatusOK, ResultSuccess("退款处理中"))
	default:
		conf.Db.Model(&model.PurchasePoleApplicationTbl{}).Where("order_number = ?", req.OrderNumber).Updates(map[string]interface{}{
			"order_status":  "退款失败",
			"refund_reason": req.RefundReason,
		})
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 退款失败，微信返回状态："+refundStatus))
	}
}
