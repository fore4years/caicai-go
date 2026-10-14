package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ StaffController（base /wx） ============

// StaffController 对齐 Java controller.StaffController。
type StaffController struct{}

// Selectname /wx/selectname?name= —— 对齐 StaffServiceImpl.selectname：
// 查 staff_tbl 是否存在该 name，存在返回 "ok"，否则 "fail"。
func (c *StaffController) Selectname(ctx *gin.Context) {
	name := ctx.Query("name")
	var staff model.StaffTbl
	if err := conf.Db.Where("name = ?", name).First(&staff).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess("fail"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// GetState /wx/getState?name= —— 对齐 Java getState：忽略入参，直接返回常量 "1"。
func (c *StaffController) GetState(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess("1"))
}
