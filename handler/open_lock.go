package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ OpenLockController（/openLock） ============

type OpenLockController struct{}

func (c *OpenLockController) List(ctx *gin.Context) {
	var rows []model.OpenLockTbl
	repository.FindAll(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *OpenLockController) Page(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.OpenLockTbl{}, &total)
	var rows []model.OpenLockTbl
	repository.Page(&model.OpenLockTbl{}, "create_time desc", pageOffset(current, size), size, &rows)
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

func (c *OpenLockController) GetById(ctx *gin.Context) {
	var o model.OpenLockTbl
	if err := repository.FindByID(&o, ctx.Param("id")).Error; err != nil {
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
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&o).Error == nil))
}

func (c *OpenLockController) Update(ctx *gin.Context) {
	var o model.OpenLockTbl
	if err := ctx.ShouldBindJSON(&o); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := repository.UpdateByID(&model.OpenLockTbl{}, o.ID, o)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *OpenLockController) DeleteById(ctx *gin.Context) {
	res := repository.DeleteByID(&model.OpenLockTbl{}, ctx.Param("id"))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}
