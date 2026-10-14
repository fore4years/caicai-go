package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
)

// ============ BillingRulesController（/wx，计价规则） ============

// billingRulesDTO 对应 Java domain.dto.BillingRulesDto（billing_rules 表）。
type billingRulesDTO struct {
	ListingPrice    string `json:"listingPrice,omitempty"`
	VipListingPrice string `json:"vipListingPrice,omitempty"`
	ServiceFee      string `json:"serviceFee,omitempty"`
	VipServiceFee   string `json:"vipServiceFee,omitempty"`
	Electricity     string `json:"electricity,omitempty"`
	VipElectricity  string `json:"vipElectricity,omitempty"`
	StartTime       string `json:"startTime,omitempty"`
	EndTime         string `json:"endTime,omitempty"`
}

// billingRulesColumns 显式列出 billing_rules 的列，避免 select * 与 DTO 命名不一致。
const billingRulesColumns = "listing_price, vip_listing_price, service_fee, vip_service_fee, electricity, vip_electricity, start_time, end_time"

type BillingRulesController struct{}

// GetAllBillingRules GET /wx/getAllBillingRules 对齐 Java getAllBillingRules：查询全部。
func (c *BillingRulesController) GetAllBillingRules(ctx *gin.Context) {
	var rows []billingRulesDTO
	conf.Db.Raw("SELECT " + billingRulesColumns + " FROM billing_rules").Scan(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

// GetBillingRules GET /wx/getBillingRules 对齐 Java getBillingRules：
// where startTime > start_time and end_time >= endTime order by start_time desc。
func (c *BillingRulesController) GetBillingRules(ctx *gin.Context) {
	startTime := ctx.Query("startTime")
	endTime := ctx.Query("endTime")
	var rows []billingRulesDTO
	conf.Db.Raw(
		"SELECT "+billingRulesColumns+" FROM billing_rules WHERE ? > start_time AND end_time >= ? ORDER BY start_time DESC",
		startTime, endTime,
	).Scan(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}
