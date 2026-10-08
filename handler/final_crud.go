package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ VideoController（/video） ============

type VideoController struct{}

func (c *VideoController) Save(ctx *gin.Context) {
	var v model.TabVideo
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "视频信息保存失败"))
		return
	}
	if conf.Db.Create(&v).Error != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "视频信息保存失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(v))
}

func (c *VideoController) List(ctx *gin.Context) {
	var rows []model.TabVideo
	conf.Db.Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

// ============ VideoWatchRecordController（/videoWatchRecord） ============

type VideoWatchRecordController struct{}

func (c *VideoWatchRecordController) Save(ctx *gin.Context) {
	var v model.TabVideoWatchRecord
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "视频观看记录保存失败"))
		return
	}
	// 对齐 Java VideoWatchRecordServiceImpl.saveVideoWatchRecord：状态置为已观看，观看时间置为当前。
	v.Status = true
	v.WatchTime = time.Now()
	if conf.Db.Create(&v).Error != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "视频观看记录保存失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(v))
}

func (c *VideoWatchRecordController) List(ctx *gin.Context) {
	var rows []model.TabVideoWatchRecord
	conf.Db.Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *VideoWatchRecordController) ListByOpenid(ctx *gin.Context) {
	var rows []model.TabVideoWatchRecord
	conf.Db.Where("openid = ?", ctx.Query("openid")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *VideoWatchRecordController) CountByFileId(ctx *gin.Context) {
	var n int64
	conf.Db.Model(&model.TabVideoWatchRecord{}).Where("file_id = ?", ctx.Query("fileId")).Count(&n)
	ctx.JSON(http.StatusOK, ResultSuccess(n))
}

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
	var lastRecord model.PackageRecordTbl
	conf.Db.Where("openid = ? AND status = 'paid'", req.Openid).
		Order("create_time desc").First(&lastRecord)
	if lastRecord.OrderNo != "" && !lastRecord.EndTime.IsZero() &&
		lastRecord.EndTime.After(now) && lastRecord.Duration != 0 {
		startTime = lastRecord.EndTime
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
		CreateTime: startTime,
		EndTime:    endTime,
	}
	if conf.Db.Create(&rec).Error != nil {
		return ""
	}
	return orderNo
}

// deductPackageBalance 对齐 Java UserServiceImpl.deductBalance（仅扣可用余额 balans）。
// 单条原子 UPDATE，条件写进 WHERE 防止余额扣成负数。
func deductPackageBalance(openid string, amount decimal.Decimal) bool {
	amt := amount.StringFixed(2)
	res := conf.Db.Exec(
		"UPDATE user_tbl SET balans = balans - CAST(? AS DECIMAL(10,2)) WHERE openid = ? AND balans >= CAST(? AS DECIMAL(10,2))",
		amt, openid, amt,
	)
	return res.RowsAffected > 0
}

func (c *PackageRecordController) BalancePay(ctx *gin.Context) {
	orderNo := packageOrderNo(ctx)
	var pkg model.PackageRecordTbl
	if err := conf.Db.Where("order_no = ?", orderNo).First(&pkg).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "余额支付失败"))
		return
	}
	if pkg.Status != "pending" {
		ctx.JSON(http.StatusOK, ResultError(500, "余额支付失败"))
		return
	}
	if !deductPackageBalance(pkg.Openid, pkg.Amount) {
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
	res := conf.Db.Model(&model.PackageRecordTbl{}).Where("order_no = ?", orderNo).Updates(updates)
	if res.RowsAffected <= 0 {
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
	var pkg model.PackageRecordTbl
	if err := conf.Db.Where("order_no = ?", orderNo).First(&pkg).Error; err != nil {
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
		conf.Db.Model(&model.PackageRecordTbl{}).Where("order_no = ?", orderNo).Update("status", "refunded")
		return "退款成功", ""
	case "PROCESSING", "CHANGE":
		conf.Db.Model(&model.PackageRecordTbl{}).Where("order_no = ?", orderNo).Update("status", "refunding")
		return "退款处理中", ""
	default:
		return "", "微信退款失败，返回状态：" + refundStatus
	}
}

func (c *PackageRecordController) GetLatestValidPackage(ctx *gin.Context) {
	var p model.PackageRecordTbl
	if err := conf.Db.Where("openid = ? AND status = 'paid' AND end_time > ?", ctx.Query("openid"), time.Now()).
		Order("create_time asc").First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "没有未过期的套餐记录"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(p))
}

func (c *PackageRecordController) GetAllValidPackages(ctx *gin.Context) {
	var rows []model.PackageRecordTbl
	conf.Db.Where("openid = ? AND status = 'paid' AND end_time > ?", ctx.Query("openid"), time.Now()).
		Order("create_time asc").Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

// GetMaxPowerFromRecentOrders 对齐 Java ReOrderPowerMapper.getMaxPowerByOpenidInWeek。
func (c *PackageRecordController) GetMaxPowerFromRecentOrders(ctx *gin.Context) {
	var maxPower decimal.Decimal
	conf.Db.Raw(`SELECT COALESCE(MAX(re_order_electronic_tbl.max_power), 0)
		FROM order_tbl
		LEFT JOIN re_order_electronic_tbl ON order_tbl.orderid = re_order_electronic_tbl.order_id
		WHERE order_tbl.openid = ?
		  AND order_tbl.consumption_type = '二轮车充电'
		  AND order_tbl.close_time >= DATE_SUB(NOW(), INTERVAL 7 DAY)`, ctx.Query("openid")).Scan(&maxPower)
	ctx.JSON(http.StatusOK, ResultSuccess(maxPower))
}

// ============ CommunityApplyController（/communityApply） ============

type CommunityApplyController struct{}

func (c *CommunityApplyController) Add(ctx *gin.Context) {
	jsonStr := ctx.PostForm("communityApply")
	if jsonStr == "" {
		jsonStr = ctx.Query("communityApply")
	}
	var a model.CommunityApplyTbl
	if jsonStr != "" {
		if err := json.Unmarshal([]byte(jsonStr), &a); err != nil {
			ctx.JSON(http.StatusOK, ResultError(500, "添加小区申请失败"))
			return
		}
	} else if err := ctx.ShouldBindJSON(&a); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "添加小区申请失败"))
		return
	}
	// 对齐 Java CommunityApplyServiceImpl.addCommunityApply：申请时间置为当前时间。
	a.ApplyTime = time.Now()
	if conf.Db.Create(&a).Error != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "添加小区申请失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ ReceptaclePowerController（/receptacle/power） ============

type ReceptaclePowerController struct{}

func (c *ReceptaclePowerController) GetPowerByPid(ctx *gin.Context) {
	var rows []model.ReceptaclePowerTbl
	conf.Db.Where("pid = ?", ctx.Query("pid")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *ReceptaclePowerController) GetPowerByReceptacleId(ctx *gin.Context) {
	var rows []model.ReceptaclePowerTbl
	conf.Db.Where("receptacle_id = ?", ctx.Query("ReceptacleId")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

// ============ CosController（/cos） ============

type CosController struct{}

func (c *CosController) Sts(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{})
}
