package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
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

func (c *PackageRecordController) Save(ctx *gin.Context) {
	var req struct {
		Openid  string          `json:"openid"`
		Amount  decimal.Decimal `json:"amount"`
		Package string          `json:"package"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "套餐充值订单创建失败"))
		return
	}
	orderNo := "PK" + time.Now().Format("20060102150405") + strconv.Itoa(int(time.Now().UnixNano()%1000))
	rec := model.PackageRecordTbl{
		Openid:  req.Openid,
		OrderNo: orderNo,
		Amount:  req.Amount,
		Status:  "待支付",
	}
	if conf.Db.Create(&rec).Error != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "套餐充值订单创建失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(orderNo))
}

func (c *PackageRecordController) BalancePay(ctx *gin.Context) {
	res := conf.Db.Model(&model.PackageRecordTbl{}).Where("order_no = ?", ctx.Query("orderNo")).Update("status", "已支付")
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PackageRecordController) Refund(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess("refund-stub"))
}

func (c *PackageRecordController) GetLatestValidPackage(ctx *gin.Context) {
	var p model.PackageRecordTbl
	if err := conf.Db.Where("openid = ? AND status = '已支付'", ctx.Query("openid")).Order("create_time desc").First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "没有未过期的套餐记录"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(p))
}

func (c *PackageRecordController) GetAllValidPackages(ctx *gin.Context) {
	var rows []model.PackageRecordTbl
	conf.Db.Where("openid = ? AND status = '已支付'", ctx.Query("openid")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *PackageRecordController) GetMaxPowerFromRecentOrders(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(0))
}

// ============ CommunityApplyController（/communityApply） ============

type CommunityApplyController struct{}

func (c *CommunityApplyController) Add(ctx *gin.Context) {
	jsonStr := ctx.PostForm("communityApply")
	var a model.CommunityApplyTbl
	if jsonStr != "" {
		_ = json.Unmarshal([]byte(jsonStr), &a)
	} else if err := ctx.ShouldBindJSON(&a); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "添加小区申请失败"))
		return
	}
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
