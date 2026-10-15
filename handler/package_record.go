package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ PackageRecordController（/packageRecord） ============

type PackageRecordController struct{}

// packageRecordRequest 对应 Java PackageRecordRequest。
type packageRecordRequest struct {
	Openid   string          `json:"openid"`
	Phone    string          `json:"phone"`
	Amount   decimal.Decimal `json:"amount"`
	Days     int32           `json:"days"`
	Duration int32           `json:"duration"`
	MaxPower int32           `json:"maxPower"`
}

// packageOrderNo 读取 orderNo 参数（对齐 Java @RequestParam，支持 query/form）。
func packageOrderNo(ctx *gin.Context) string {
	if v := ctx.Query("orderNo"); v != "" {
		return v
	}
	return ctx.PostForm("orderNo")
}

func (c *PackageRecordController) Save(ctx *gin.Context) {
	var req packageRecordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "套餐充值订单创建失败"))
		return
	}
	orderNo := c.createPackageRecord(req)
	if orderNo == "" {
		ctx.JSON(http.StatusOK, ResultError(500, "套餐充值订单创建失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(orderNo))
}

// createPackageRecord 对齐 Java PackageRecordServiceImpl.createPackageRecord。
func (c *PackageRecordController) createPackageRecord(req packageRecordRequest) string {
	orderNo := "PR" + time.Now().Format("20060102") + fmt.Sprintf("%06d", rand.Intn(1000000))
	now := time.Now()
	startTime := now
	var endTime time.Time

	// 查询最近一条已支付套餐：若尚未过期且时长非 0，则在其结束时间上累加天数。
	lastRecord := repository.LatestPaidPackageRecord(req.Openid)
	if lastRecord.OrderNo != "" && !lastRecord.EndTime.IsZero() &&
		lastRecord.EndTime.After(now) && lastRecord.Duration != 0 {
		startTime = timeOrZero(lastRecord.EndTime)
		endTime = startTime.AddDate(0, 0, int(req.Days))
	} else {
		endTime = now.AddDate(0, 0, int(req.Days))
	}

	rec := model.PackageRecordTbl{
		OrderNo:    orderNo,
		Openid:     req.Openid,
		Phone:      req.Phone,
		Amount:     req.Amount,
		Days:       req.Days,
		Duration:   req.Duration,
		MaxPower:   req.MaxPower,
		Status:     "pending",
		CreateTime: timePtr(startTime),
		EndTime:    timePtr(endTime),
	}
	if repository.Create(&rec).Error != nil {
		return ""
	}
	return orderNo
}

func (c *PackageRecordController) BalancePay(ctx *gin.Context) {
	orderNo := packageOrderNo(ctx)
	pkg, err := repository.PackageRecordByOrderNo(orderNo)
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "余额支付失败"))
		return
	}
	if pkg.Status != "pending" {
		ctx.JSON(http.StatusOK, ResultError(500, "余额支付失败"))
		return
	}
	if !repository.DeductPackageBalance(pkg.Openid, pkg.Amount) {
		ctx.JSON(http.StatusOK, ResultError(500, "余额支付失败"))
		return
	}
	updates := map[string]interface{}{
		"status":         "paid",
		"transaction_id": "余额付",
	}
	if pkg.EndTime.IsZero() && !pkg.CreateTime.IsZero() {
		updates["end_time"] = pkg.CreateTime.AddDate(0, 0, int(pkg.Days))
	}
	rowsAffected := repository.UpdatePackageRecordByOrderNo(orderNo, updates)
	if rowsAffected <= 0 {
		ctx.JSON(http.StatusOK, ResultError(500, "余额支付失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

func (c *PackageRecordController) Refund(ctx *gin.Context) {
	result, errMsg := c.processRefund(packageOrderNo(ctx))
	if errMsg != "" {
		ctx.JSON(http.StatusOK, ResultError(500, "微信退款异常: "+errMsg))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(result))
}

// processRefund 对齐 Java PackageRecordServiceImpl.processRefund。
func (c *PackageRecordController) processRefund(orderNo string) (string, string) {
	pkg, err := repository.PackageRecordByOrderNo(orderNo)
	if err != nil {
		return "", "订单不存在"
	}
	if pkg.Status != "paid" {
		return "", "订单状态不正确，只有已支付的订单才能退款"
	}
	if pkg.TransactionID == "余额付" {
		return "", "该订单使用余额支付，不支持微信退款"
	}

	refundOrderID := "REFUND_PR_" + fmt.Sprintf("%d", time.Now().UnixMilli())
	refundFee := int(pkg.Amount.Mul(decimal.NewFromInt(100)).IntPart())
	totalFee := refundFee
	refundStatus, err := refundWxOrder(pkg.TransactionID, refundOrderID, refundFee, totalFee)
	if err != nil {
		return "", "微信退款失败: " + err.Error()
	}
	switch refundStatus {
	case "SUCCESS":
		repository.UpdatePackageRecordStatus(orderNo, "refunded")
		return "退款成功", ""
	case "PROCESSING", "CHANGE":
		repository.UpdatePackageRecordStatus(orderNo, "refunding")
		return "退款处理中", ""
	default:
		return "", "微信退款失败，返回状态：" + refundStatus
	}
}

func (c *PackageRecordController) GetLatestValidPackage(ctx *gin.Context) {
	p, err := repository.LatestValidPackageByOpenid(ctx.Query("openid"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "没有未过期的套餐记录"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(p))
}

func (c *PackageRecordController) GetAllValidPackages(ctx *gin.Context) {
	rows := repository.ValidPackagesByOpenid(ctx.Query("openid"))
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

// GetMaxPowerFromRecentOrders 对齐 Java ReOrderPowerMapper.getMaxPowerByOpenidInWeek。
func (c *PackageRecordController) GetMaxPowerFromRecentOrders(ctx *gin.Context) {
	maxPower := repository.MaxPowerFromRecentOrders(ctx.Query("openid"))
	ctx.JSON(http.StatusOK, ResultSuccess(maxPower))
}
