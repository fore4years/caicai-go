package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

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
	a.ApplyTime = timePtr(time.Now())
	if conf.Db.Create(&a).Error != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "添加小区申请失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}
