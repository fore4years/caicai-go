package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ VideoController（/video） ============

type VideoController struct{}

func (c *VideoController) Save(ctx *gin.Context) {
	var v model.TabVideo
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "视频信息保存失败"))
		return
	}
	if repository.Create(&v).Error != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "视频信息保存失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(v))
}

func (c *VideoController) List(ctx *gin.Context) {
	var rows []model.TabVideo
	repository.FindAll(&rows)
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
	v.WatchTime = timePtr(time.Now())
	if repository.Create(&v).Error != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "视频观看记录保存失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(v))
}

func (c *VideoWatchRecordController) List(ctx *gin.Context) {
	var rows []model.TabVideoWatchRecord
	repository.FindAll(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *VideoWatchRecordController) ListByOpenid(ctx *gin.Context) {
	rows := repository.VideoWatchRecordsByOpenid(ctx.Query("openid"))
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *VideoWatchRecordController) CountByFileId(ctx *gin.Context) {
	n := repository.CountVideoWatchRecordsByFileID(ctx.Query("fileId"))
	ctx.JSON(http.StatusOK, ResultSuccess(n))
}
