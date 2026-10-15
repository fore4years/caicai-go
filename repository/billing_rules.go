package repository

import "caicai-go/conf"

// BillingRules 对应 Java domain.dto.BillingRulesDto（billing_rules 表）。
type BillingRules struct {
	ListingPrice    string `json:"listingPrice,omitempty"`
	VipListingPrice string `json:"vipListingPrice,omitempty"`
	ServiceFee      string `json:"serviceFee,omitempty"`
	VipServiceFee   string `json:"vipServiceFee,omitempty"`
	Electricity     string `json:"electricity,omitempty"`
	VipElectricity  string `json:"vipElectricity,omitempty"`
	StartTime       string `json:"startTime,omitempty"`
	EndTime         string `json:"endTime,omitempty"`
}

const billingRulesColumns = "listing_price, vip_listing_price, service_fee, vip_service_fee, electricity, vip_electricity, start_time, end_time"

// AllBillingRules 查询全部计价规则。
func AllBillingRules() []BillingRules {
	var rows []BillingRules
	conf.Db.Raw("SELECT " + billingRulesColumns + " FROM billing_rules").Scan(&rows)
	return rows
}

// BillingRulesByTime 按时间查询计价规则。
func BillingRulesByTime(startTime, endTime string) []BillingRules {
	var rows []BillingRules
	conf.Db.Raw(
		"SELECT "+billingRulesColumns+" FROM billing_rules WHERE ? > start_time AND end_time >= ? ORDER BY start_time DESC",
		startTime, endTime,
	).Scan(&rows)
	return rows
}
