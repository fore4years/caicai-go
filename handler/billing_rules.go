package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/repository"
)

// ============ BillingRulesController（/wx，计价规则） ============

type BillingRulesController struct{}

// GetAllBillingRules GET /wx/getAllBillingRules 对齐 Java getAllBillingRules：查询全部。
func (c *BillingRulesController) GetAllBillingRules(ctx *gin.Context) {
	rows := repository.AllBillingRules()
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

// GetBillingRules GET /wx/getBillingRules 对齐 Java getBillingRules：
// where startTime > start_time and end_time >= endTime order by start_time desc。
func (c *BillingRulesController) GetBillingRules(ctx *gin.Context) {
	startTime := ctx.Query("startTime")
	endTime := ctx.Query("endTime")
	rows := repository.BillingRulesByTime(startTime, endTime)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}
