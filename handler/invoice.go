package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ UserInvoiceController（/invoice，用户侧） ============

type UserInvoiceController struct{}

// invoiceTitleDTO 对应 Java domain.InvoiceTitle（驼峰）。
type invoiceTitleDTO struct {
	ID          int32  `json:"id"`
	TitleName   string `json:"titleName,omitempty"`
	FarmTax     string `json:"farmTax,omitempty"`
	FarmAddress string `json:"farmAddress,omitempty"`
	FarmPhone   string `json:"farmPhone,omitempty"`
	FarmBank    string `json:"farmBank,omitempty"`
	OpenBank    string `json:"openBank,omitempty"`
	Email       string `json:"email,omitempty"`
	Openid      string `json:"openid,omitempty"`
}

func invoiceTitleToDTO(t model.InvoiceTitle) invoiceTitleDTO {
	return invoiceTitleDTO{
		ID:          t.ID,
		TitleName:   t.TitleName,
		FarmTax:     t.FarmTax,
		FarmAddress: t.FarmAddress,
		FarmPhone:   t.FarmPhone,
		FarmBank:    t.FarmBank,
		OpenBank:    t.OpenBank,
		Email:       t.Email,
		Openid:      t.Openid,
	}
}

// GetByInvoiceId 对应 Java InvoiceController.getByInvoiceId。
func (c *UserInvoiceController) GetByInvoiceId(ctx *gin.Context) {
	var inv model.Invoice
	if err := repository.FindByID(&inv, ctx.Query("id")).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(invoiceToDTO(inv)))
}

// UpdateById 对应 Java InvoiceController.updateById：state=1、invoice_id=titleId、apply_time=now。
func (c *UserInvoiceController) UpdateById(ctx *gin.Context) {
	res := repository.UpdateByID(&model.Invoice{}, ctx.Query("id"), map[string]interface{}{
		"state":      1,
		"invoice_id": ctx.Query("titleId"),
		"apply_time": time.Now(),
	})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

// GetInvoiceByOpenid 对应 Java InvoiceController.getInvoiceByOpenid。
func (c *UserInvoiceController) GetInvoiceByOpenid(ctx *gin.Context) {
	rows := repository.InvoicesByOpenid(ctx.Query("openid"))
	list := make([]invoiceDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, invoiceToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// Add 对应 Java InvoiceController.add：新增发票抬头，openid 取自当前登录用户。
func (c *UserInvoiceController) Add(ctx *gin.Context) {
	var req invoiceTitleDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	t := model.InvoiceTitle{
		TitleName:   req.TitleName,
		FarmTax:     req.FarmTax,
		FarmAddress: req.FarmAddress,
		FarmPhone:   req.FarmPhone,
		FarmBank:    req.FarmBank,
		OpenBank:    req.OpenBank,
		Email:       req.Email,
		Openid:      ctx.GetString("openid"),
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&t).Error == nil))
}

// GetByOpenid 对应 Java InvoiceController.getByOpenid：按当前登录用户 openid 查抬头。
func (c *UserInvoiceController) GetByOpenid(ctx *gin.Context) {
	rows := repository.InvoiceTitlesByOpenid(ctx.GetString("openid"))
	list := make([]invoiceTitleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, invoiceTitleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetById 对应 Java InvoiceController.getById。
func (c *UserInvoiceController) GetById(ctx *gin.Context) {
	var t model.InvoiceTitle
	if err := repository.FindByID(&t, ctx.Query("id")).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(invoiceTitleToDTO(t)))
}

// DeleteById 对应 Java InvoiceController.deleteById：先置发票 state=0 再删抬头。
func (c *UserInvoiceController) DeleteById(ctx *gin.Context) {
	id := ctx.Query("id")
	repository.UpdateInvoiceStateByInvoiceID(id, 0)
	res := repository.DeleteByID(&model.InvoiceTitle{}, id)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

// UpdetaById 对应 Java InvoiceController.updetaById：按 id 编辑发票抬头。
func (c *UserInvoiceController) UpdetaById(ctx *gin.Context) {
	var req invoiceTitleDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	res := repository.UpdateByID(&model.InvoiceTitle{}, req.ID, invoiceTitleUpdates(req))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

// AddInvoice 对应 Java InvoiceController.addInvoice：实际为按 id 更新发票抬头。
func (c *UserInvoiceController) AddInvoice(ctx *gin.Context) {
	var req invoiceTitleDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	res := repository.UpdateByID(&model.InvoiceTitle{}, req.ID, invoiceTitleUpdates(req))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

// invoiceTitleUpdates 将请求体中的非空抬头字段转换为更新 map（对齐 MyBatis-Plus 非空更新策略）。
func invoiceTitleUpdates(req invoiceTitleDTO) map[string]interface{} {
	updates := map[string]interface{}{}
	if req.TitleName != "" {
		updates["title_name"] = req.TitleName
	}
	if req.FarmTax != "" {
		updates["farm_tax"] = req.FarmTax
	}
	if req.FarmAddress != "" {
		updates["farm_address"] = req.FarmAddress
	}
	if req.FarmPhone != "" {
		updates["farm_phone"] = req.FarmPhone
	}
	if req.FarmBank != "" {
		updates["farm_bank"] = req.FarmBank
	}
	if req.OpenBank != "" {
		updates["open_bank"] = req.OpenBank
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Openid != "" {
		updates["openid"] = req.Openid
	}
	return updates
}
