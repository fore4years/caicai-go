package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ VersionController（/version） ============

type VersionController struct{}

func (c *VersionController) List(ctx *gin.Context) {
	var rows []model.VersionTbl
	repository.FindAll(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *VersionController) Page(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.VersionTbl{}, &total)
	var rows []model.VersionTbl
	repository.Page(&model.VersionTbl{}, "create_time desc", pageOffset(current, size), size, &rows)
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

func (c *VersionController) GetById(ctx *gin.Context) {
	var v model.VersionTbl
	if err := repository.FindByID(&v, ctx.Param("id")).Error; err != nil {
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
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&v).Error == nil))
}

func (c *VersionController) Update(ctx *gin.Context) {
	var v model.VersionTbl
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := repository.UpdateByID(&model.VersionTbl{}, v.ID, v)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *VersionController) DeleteById(ctx *gin.Context) {
	res := repository.DeleteByID(&model.VersionTbl{}, ctx.Param("id"))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}
