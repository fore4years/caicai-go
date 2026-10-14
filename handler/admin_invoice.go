package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// invoiceDTO 对应 Java domain.Invoice（invoice，驼峰）。
type invoiceDTO struct {
	ID          int32           `json:"id"`
	InvoiceType string          `json:"invoiceType,omitempty"`
	InvoiceDate *LocalDateTime  `json:"invoiceDate,omitempty"`
	FarmTax     string          `json:"farmTax,omitempty"`
	FarmName    string          `json:"farmName,omitempty"`
	FarmAddress string          `json:"farmAddress,omitempty"`
	InvoiceCode string          `json:"invoiceCode,omitempty"`
	InvoiceNum  string          `json:"invoiceNum,omitempty"`
	Amount      decimal.Decimal `json:"amount,omitempty"`
	TaxAmount   decimal.Decimal `json:"taxAmount,omitempty"`
	InvoiceID   int32           `json:"invoiceId,omitempty"`
	State       int32           `json:"state,omitempty"`
	Openid      string          `json:"openid,omitempty"`
	ApplyTime   *LocalDateTime  `json:"applyTime,omitempty"`
}

func invoiceToDTO(i model.Invoice) invoiceDTO {
	return invoiceDTO{
		ID:          i.ID,
		InvoiceType: i.InvoiceType,
		InvoiceDate: timeToLocal(i.InvoiceDate),
		FarmTax:     i.FarmTax,
		FarmName:    i.FarmName,
		FarmAddress: i.FarmAddress,
		InvoiceCode: i.InvoiceCode,
		InvoiceNum:  i.InvoiceNum,
		Amount:      i.Amount,
		TaxAmount:   i.TaxAmount,
		InvoiceID:   i.InvoiceID,
		State:       i.State,
		Openid:      i.Openid,
		ApplyTime:   timeToLocal(i.ApplyTime),
	}
}

// invoiceDetailDTO 对应 Java System.domain.InvoiceDetailVo（发票明细视图对象，含关联的手机号与抬头信息）。
type invoiceDetailDTO struct {
	ID          int32           `json:"id"`
	Openid      string          `json:"openid,omitempty"`
	Phone       string          `json:"phone,omitempty"`
	InvoiceType string          `json:"invoiceType,omitempty"`
	InvoiceDate *LocalDateTime  `json:"invoiceDate,omitempty"`
	ApplyTime   *LocalDateTime  `json:"applyTime,omitempty"`
	TitleName   string          `json:"titleName,omitempty"`
	Email       string          `json:"email,omitempty"`
	Amount      decimal.Decimal `json:"amount,omitempty"`
	State       int32           `json:"state,omitempty"`
}

// AdminInvoiceController 对齐 Java System.controller.InvoiceDetailCon。
type AdminInvoiceController struct{}

// List GET /system/invoice/list
// 对齐 Java InvoiceDetailSerImpl.getInvoiceList：仅查 state IN (1,2)，手机号在分页后按 openid 过滤，
// 并按 openid 批量关联 user_tbl 与 invoice_title 组装 InvoiceDetailVo。
func (a *AdminInvoiceController) List(c *gin.Context) {
	current, size := parsePage(c)
	openid := c.Query("openid")
	phone := c.Query("phone")

	emptyResult := func() gin.H {
		return gin.H{"list": []invoiceDetailDTO{}, "total": int64(0), "pageNum": current, "pageSize": size, "pages": int64(0)}
	}

	// 1. 查询 state IN (1,2) 的发票，按发票日期降序分页
	base := conf.Db.Model(&model.Invoice{}).Where("state IN ?", []int32{1, 2})
	if openid != "" {
		base = base.Where("openid = ?", openid)
	}

	var total int64
	base.Count(&total)

	var invoices []model.Invoice
	base.Order("invoice_date desc").Offset(pageOffset(current, size)).Limit(size).Find(&invoices)

	if len(invoices) == 0 {
		c.JSON(http.StatusOK, ResultSuccess(emptyResult()))
		return
	}

	// 2. 收集当前页所有 openid
	openidSet := make(map[string]bool, len(invoices))
	for _, inv := range invoices {
		openidSet[inv.Openid] = true
	}
	openids := make([]string, 0, len(openidSet))
	for o := range openidSet {
		openids = append(openids, o)
	}

	// 3. 手机号筛选（分页之后按 openid 过滤）
	if phone != "" {
		var users []model.UserTbl
		conf.Db.Where("phone = ?", phone).Find(&users)
		if len(users) == 0 {
			c.JSON(http.StatusOK, ResultSuccess(emptyResult()))
			return
		}
		valid := make(map[string]bool, len(users))
		for _, u := range users {
			valid[u.Openid] = true
		}
		var filtered []model.Invoice
		for _, inv := range invoices {
			if valid[inv.Openid] {
				filtered = append(filtered, inv)
			}
		}
		invoices = filtered
	}

	// 4. 批量查询用户信息（openid → phone）
	userMap := make(map[string]model.UserTbl)
	if len(openids) > 0 {
		var users []model.UserTbl
		conf.Db.Where("openid IN ?", openids).Find(&users)
		for _, u := range users {
			if u.Openid != "" {
				userMap[u.Openid] = u
			}
		}
	}

	// 5. 批量查询发票抬头（openid → titleName/email）
	titleMap := make(map[string]model.InvoiceTitle)
	if len(openids) > 0 {
		var titles []model.InvoiceTitle
		conf.Db.Where("openid IN ?", openids).Find(&titles)
		for _, t := range titles {
			if t.Openid != "" {
				titleMap[t.Openid] = t
			}
		}
	}

	// 6. 组装 InvoiceDetailVo
	list := make([]invoiceDetailDTO, 0, len(invoices))
	for _, inv := range invoices {
		dto := invoiceDetailDTO{
			ID:          inv.ID,
			Openid:      inv.Openid,
			InvoiceType: inv.InvoiceType,
			InvoiceDate: timeToLocal(inv.InvoiceDate),
			ApplyTime:   timeToLocal(inv.ApplyTime),
			Amount:      inv.Amount,
			State:       inv.State,
		}
		if u, ok := userMap[inv.Openid]; ok {
			dto.Phone = u.Phone
		}
		if t, ok := titleMap[inv.Openid]; ok {
			dto.TitleName = t.TitleName
			dto.Email = t.Email
		}
		list = append(list, dto)
	}

	// 7. 构建返回结果
	pages := int64(0)
	if total > 0 {
		pages = (total + int64(size) - 1) / int64(size)
	}
	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"list":     list,
		"total":    total,
		"pageNum":  current,
		"pageSize": size,
		"pages":    pages,
	}))
}

// Process PUT /system/invoice/process?id=
// 对齐 Java InvoiceDetailSerImpl.processInvoice：不存在或已处理(state=2)时返回失败。
func (a *AdminInvoiceController) Process(c *gin.Context) {
	id := c.Query("id")
	var inv model.Invoice
	if err := conf.Db.Where("id = ?", id).First(&inv).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(500, "发票处理失败"))
		return
	}
	if inv.State == 2 {
		c.JSON(http.StatusOK, ResultError(500, "发票处理失败"))
		return
	}
	res := conf.Db.Model(&model.Invoice{}).Where("id = ?", id).Update("state", 2)
	if res.RowsAffected > 0 {
		c.JSON(http.StatusOK, ResultSuccess("发票处理成功"))
		return
	}
	c.JSON(http.StatusOK, ResultError(500, "发票处理失败"))
}

// BatchDelete POST /system/invoice/batchDelete
func (a *AdminInvoiceController) BatchDelete(c *gin.Context) {
	var ids []int32
	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
		return
	}
	if len(ids) == 0 {
		c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
		return
	}
	res := conf.Db.Where("id IN ?", ids).Delete(&model.Invoice{})
	if res.RowsAffected > 0 {
		c.JSON(http.StatusOK, ResultSuccess("批量删除成功"))
		return
	}
	c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
}

// CheckRecentApply GET /system/invoice/checkRecentApply
func (a *AdminInvoiceController) CheckRecentApply(c *gin.Context) {
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	var n int64
	conf.Db.Model(&model.Invoice{}).
		Where("state = 1 AND apply_time >= ? AND apply_time <= ?", twoDaysAgo, time.Now()).
		Count(&n)
	c.JSON(http.StatusOK, ResultSuccess(n > 0))
}
