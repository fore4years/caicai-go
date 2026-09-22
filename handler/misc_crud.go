package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ ConfigController（/config） ============

type ConfigController struct{}

func (c *ConfigController) Save(ctx *gin.Context) {
	var cfg model.ConfigTbl
	if err := ctx.ShouldBind(&cfg); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Save(&cfg).Error == nil))
}

// ============ VersionController（/version） ============

type VersionController struct{}

func (c *VersionController) List(ctx *gin.Context) {
	var rows []model.VersionTbl
	conf.Db.Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *VersionController) Page(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.VersionTbl{}).Count(&total)
	var rows []model.VersionTbl
	conf.Db.Model(&model.VersionTbl{}).Order("create_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

func (c *VersionController) GetById(ctx *gin.Context) {
	var v model.VersionTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&v).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(v))
}

func (c *VersionController) Save(ctx *gin.Context) {
	var v model.VersionTbl
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&v).Error == nil))
}

func (c *VersionController) Update(ctx *gin.Context) {
	var v model.VersionTbl
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Model(&model.VersionTbl{}).Where("id = ?", v.ID).Updates(v)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *VersionController) DeleteById(ctx *gin.Context) {
	res := conf.Db.Where("id = ?", ctx.Param("id")).Delete(&model.VersionTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ OpenLockController（/openLock） ============

type OpenLockController struct{}

func (c *OpenLockController) List(ctx *gin.Context) {
	var rows []model.OpenLockTbl
	conf.Db.Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *OpenLockController) Page(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.OpenLockTbl{}).Count(&total)
	var rows []model.OpenLockTbl
	conf.Db.Model(&model.OpenLockTbl{}).Order("create_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

func (c *OpenLockController) GetById(ctx *gin.Context) {
	var o model.OpenLockTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&o).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(o))
}

func (c *OpenLockController) Save(ctx *gin.Context) {
	var o model.OpenLockTbl
	if err := ctx.ShouldBindJSON(&o); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&o).Error == nil))
}

func (c *OpenLockController) Update(ctx *gin.Context) {
	var o model.OpenLockTbl
	if err := ctx.ShouldBindJSON(&o); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Model(&model.OpenLockTbl{}).Where("id = ?", o.ID).Updates(o)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *OpenLockController) DeleteById(ctx *gin.Context) {
	res := conf.Db.Where("id = ?", ctx.Param("id")).Delete(&model.OpenLockTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ DriverUseController（/driverUse） ============

type DriverUseController struct{}

func (c *DriverUseController) Save(ctx *gin.Context) {
	openid := ctx.Query("openid")
	rec := model.DriveruseTbl{
		Openid:   openid,
		EveryDay: time.Now(),
	}
	conf.Db.Create(&rec)
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ BillingRulesController（/wx，计价规则） ============

type BillingRulesController struct{}

func (c *BillingRulesController) GetAllBillingRules(ctx *gin.Context) {
	var rows []model.TabPrice
	conf.Db.Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *BillingRulesController) GetBillingRules(ctx *gin.Context) {
	var rows []model.TabPrice
	conf.Db.Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}
