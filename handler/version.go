package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

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
