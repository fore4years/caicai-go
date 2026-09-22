package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ UserController 其余端点（挂在 OrderController 上，base /user） ============

func (o *OrderController) GetUserInfoByOpenId(c *gin.Context) {
	var u model.UserTbl
	if err := conf.Db.Where("openid = ?", c.Param("openid")).First(&u).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(userToDTO(u)))
}

func (o *OrderController) UpdateUserInfo(c *gin.Context) {
	var req struct {
		Openid   string `json:"openid"`
		NickName string `json:"nickName"`
		PlateNum string `json:"plateNum"`
		IDNumber string `json:"idNumber"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	updates := map[string]interface{}{}
	if req.NickName != "" {
		updates["nick_name"] = req.NickName
	}
	if req.PlateNum != "" {
		updates["plate_num"] = req.PlateNum
	}
	if req.IDNumber != "" {
		updates["id_number"] = req.IDNumber
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	openid := req.Openid
	if openid == "" {
		openid = c.GetString("openid")
	}
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Updates(updates)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (o *OrderController) UpdatePhone(c *gin.Context) {
	var req struct {
		Openid string `json:"openid"`
		Phone  string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(""))
		return
	}
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", req.Openid).Update("phone", req.Phone)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (o *OrderController) RefToken(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(gin.H{}))
}

func (o *OrderController) UpdateUserIdEntity(c *gin.Context) {
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", c.Query("openid")).Update("id_entity", c.Query("idEntity"))
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (o *OrderController) UpdateUserStatus(c *gin.Context) {
	openid := c.GetString("openid")
	conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Updates(map[string]interface{}{
		"is_steer":     c.Query("isSteer"),
		"is_procedure": c.Query("isProcedure"),
		"is_login":     c.Query("isLogin"),
	})
	c.JSON(http.StatusOK, ResultSuccess(""))
}

// ============ UserInvoiceController（/invoice，用户侧） ============

type UserInvoiceController struct{}

func (c *UserInvoiceController) GetByInvoiceId(ctx *gin.Context) {
	var inv model.Invoice
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&inv).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(invoiceToDTO(inv)))
}

func (c *UserInvoiceController) UpdateById(ctx *gin.Context) {
	res := conf.Db.Model(&model.Invoice{}).Where("id = ?", ctx.Query("id")).Updates(map[string]interface{}{
		"state":      1,
		"invoice_id": ctx.Query("titleId"),
		"apply_time": time.Now(),
	})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

func (c *UserInvoiceController) GetInvoiceByOpenid(ctx *gin.Context) {
	var rows []model.Invoice
	conf.Db.Where("openid = ?", ctx.Query("openid")).Find(&rows)
	list := make([]invoiceDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, invoiceToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *UserInvoiceController) Add(ctx *gin.Context) {
	var t model.InvoiceTitle
	if err := ctx.ShouldBindJSON(&t); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	t.Openid = ctx.GetString("openid")
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&t).Error == nil))
}

func (c *UserInvoiceController) GetByOpenid(ctx *gin.Context) {
	var rows []model.InvoiceTitle
	conf.Db.Where("openid = ?", ctx.GetString("openid")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *UserInvoiceController) GetById(ctx *gin.Context) {
	var t model.InvoiceTitle
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&t).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(t))
}

func (c *UserInvoiceController) DeleteById(ctx *gin.Context) {
	id := ctx.Query("id")
	conf.Db.Model(&model.Invoice{}).Where("invoice_id = ?", id).Update("state", 0)
	res := conf.Db.Where("id = ?", id).Delete(&model.InvoiceTitle{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

func (c *UserInvoiceController) UpdetaById(ctx *gin.Context) {
	var t model.InvoiceTitle
	if err := ctx.ShouldBindJSON(&t); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	res := conf.Db.Model(&model.InvoiceTitle{}).Where("id = ?", t.ID).Updates(t)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

func (c *UserInvoiceController) AddInvoice(ctx *gin.Context) {
	var t model.InvoiceTitle
	if err := ctx.ShouldBindJSON(&t); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	res := conf.Db.Model(&model.InvoiceTitle{}).Where("id = ?", t.ID).Updates(t)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

// ============ PurchasePoleApplicationController（/purchasePoleApplication） ============

type PurchasePoleApplicationController struct{}

func (c *PurchasePoleApplicationController) UpdateOrderStatus(ctx *gin.Context) {
	res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
		Where("order_number = ?", ctx.Query("orderNumber")).
		Update("order_status", ctx.Query("orderStatus"))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PurchasePoleApplicationController) GetByOrderStatus(ctx *gin.Context) {
	var rows []model.PurchasePoleApplicationTbl
	conf.Db.Where("order_status = ?", ctx.Param("orderStatus")).Find(&rows)
	list := make([]purchasePoleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, purchasePoleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PurchasePoleApplicationController) Create(ctx *gin.Context) {
	var p model.PurchasePoleApplicationTbl
	if err := ctx.ShouldBindJSON(&p); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "购桩申请创建失败"))
		return
	}
	p.OrderNumber = "PP" + time.Now().Format("20060102150405")
	p.OrderStatus = "待支付"
	if conf.Db.Create(&p).Error != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "购桩申请创建失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(p.OrderNumber))
}

func (c *PurchasePoleApplicationController) GetCurrentUserApplication(ctx *gin.Context) {
	c.GetByOpenid(ctx)
}

func (c *PurchasePoleApplicationController) GetByOpenid(ctx *gin.Context) {
	openid := ctx.Param("openid")
	if openid == "" {
		openid = ctx.GetString("openid")
	}
	var rows []model.PurchasePoleApplicationTbl
	conf.Db.Where("openid = ?", openid).Order("apply_time desc").Find(&rows)
	list := make([]purchasePoleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, purchasePoleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PurchasePoleApplicationController) DeleteByOrderNumber(ctx *gin.Context) {
	res := conf.Db.Where("order_number = ?", ctx.Param("orderNumber")).Delete(&model.PurchasePoleApplicationTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PurchasePoleApplicationController) Pay(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{}))
}

func (c *PurchasePoleApplicationController) UpdatePaymentInfo(ctx *gin.Context) {
	res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
		Where("order_number = ?", ctx.Query("orderNumber")).
		Updates(map[string]interface{}{"transaction_id": ctx.Query("transactionId"), "pay_time": time.Now()})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PurchasePoleApplicationController) GetByOrderNumber(ctx *gin.Context) {
	var p model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", ctx.Param("orderNumber")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "未找到购桩申请信息"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(purchasePoleToDTO(p)))
}

func (c *PurchasePoleApplicationController) ApplyAndProcessRefund(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess("refund-stub"))
}
