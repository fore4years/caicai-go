package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

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
