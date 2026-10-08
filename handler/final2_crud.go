package handler

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/middleware"
	"caicai-go/model"
)

// ============ UserController 其余端点（挂在 OrderController 上，base /user） ============

// GetUserInfoByOpenId 对应 Java UserController.getUserInfoByOpenId：根据 openid 查用户。
func (o *OrderController) GetUserInfoByOpenId(c *gin.Context) {
	var u model.UserTbl
	if err := conf.Db.Where("openid = ?", c.Param("openid")).First(&u).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(userToDTO(u)))
}

// UpdateUserInfo 对应 Java UserServiceImpl.updateUserInfo：
//   - 主键是 phone，更新条件为 phone = 当前登录用户手机号（JWT aud）。
//   - phone 取当前登录用户手机号。
//   - omid / openid / free_time / integral / id_card_avatar / id_card_emblem 置空，不参与更新。
func (o *OrderController) UpdateUserInfo(c *gin.Context) {
	var req struct {
		NickName     *string          `json:"nickName"`
		Province     *string          `json:"province"`
		City         *string          `json:"city"`
		PlateNum     *string          `json:"plateNum"`
		IDNumber     *string          `json:"idNumber"`
		Name         *string          `json:"name"`
		Avatar       []byte           `json:"avatar"`
		IDEntity     *string          `json:"idEntity"`
		OmEnable     *string          `json:"omEnable"`
		IsSteer      *string          `json:"isSteer"`
		IsProcedure  *string          `json:"isProcedure"`
		IsLogin      *string          `json:"isLogin"`
		YiparlOpenid *string          `json:"yiparlOpenid"`
		Balans       *decimal.Decimal `json:"balans"`
		FreezeBalans *decimal.Decimal `json:"freezeBalans"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}

	tokenPhone := c.GetString("phone")
	updates := map[string]interface{}{"phone": tokenPhone}
	if req.NickName != nil {
		updates["nick_name"] = *req.NickName
	}
	if req.Province != nil {
		updates["province"] = *req.Province
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.PlateNum != nil {
		updates["plate_num"] = *req.PlateNum
	}
	if req.IDNumber != nil {
		updates["id_number"] = *req.IDNumber
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if len(req.Avatar) > 0 {
		updates["avatar"] = req.Avatar
	}
	if req.IDEntity != nil {
		updates["id_entity"] = *req.IDEntity
	}
	if req.OmEnable != nil {
		updates["om_enable"] = *req.OmEnable
	}
	if req.IsSteer != nil {
		updates["is_steer"] = *req.IsSteer
	}
	if req.IsProcedure != nil {
		updates["is_procedure"] = *req.IsProcedure
	}
	if req.IsLogin != nil {
		updates["is_login"] = *req.IsLogin
	}
	if req.YiparlOpenid != nil {
		updates["yiparl_openid"] = *req.YiparlOpenid
	}
	if req.Balans != nil {
		updates["balans"] = *req.Balans
	}
	if req.FreezeBalans != nil {
		updates["freeze_balans"] = *req.FreezeBalans
	}

	res := conf.Db.Model(&model.UserTbl{}).Where("phone = ?", tokenPhone).Updates(updates)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// UpdatePhone 对应 Java UserServiceImpl.updatePhone：返回字符串消息。
func (o *OrderController) UpdatePhone(c *gin.Context) {
	var req struct {
		Openid string `json:"openid"`
		Phone  string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(""))
		return
	}
	if req.Phone == c.GetString("phone") {
		c.JSON(http.StatusOK, ResultSuccess("修改失败不能使用同一个手机号"))
		return
	}
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", req.Openid).Update("phone", req.Phone)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusOK, ResultSuccess("修改失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("修改成功"))
}

// RefToken 对应 Java UserServiceImpl.refToken：解析旧 token 后重新签发。
func (o *OrderController) RefToken(c *gin.Context) {
	phone, openid, name, err := middleware.ParseToken(c.Query("token"))
	if err != nil {
		c.JSON(http.StatusOK, ResultError(401, "未登录"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(getToken(phone, openid, name)))
}

// UpdateUserIdEntity 对应 Java UserServiceImpl.updateUserIdEntity：返回 "ok"/"false"。
func (o *OrderController) UpdateUserIdEntity(c *gin.Context) {
	res := conf.Db.Model(&model.UserTbl{}).
		Where("openid = ?", c.Query("openid")).
		Update("id_entity", c.Query("idEntity"))
	if res.RowsAffected == 0 {
		c.JSON(http.StatusOK, ResultSuccess("false"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("ok"))
}

// UpdateUserStatus 对应 Java UserServiceImpl.updateUserStatus：仅更新非空参数，返回 "ok"/"false"。
func (o *OrderController) UpdateUserStatus(c *gin.Context) {
	openid := c.GetString("openid")
	updates := map[string]interface{}{}
	if v := c.Query("isSteer"); v != "" {
		updates["is_steer"] = v
	}
	if v := c.Query("isProcedure"); v != "" {
		updates["is_procedure"] = v
	}
	if v := c.Query("isLogin"); v != "" {
		updates["is_login"] = v
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, ResultSuccess("false"))
		return
	}
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Updates(updates)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusOK, ResultSuccess("false"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("ok"))
}

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
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&inv).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(invoiceToDTO(inv)))
}

// UpdateById 对应 Java InvoiceController.updateById：state=1、invoice_id=titleId、apply_time=now。
func (c *UserInvoiceController) UpdateById(ctx *gin.Context) {
	res := conf.Db.Model(&model.Invoice{}).Where("id = ?", ctx.Query("id")).Updates(map[string]interface{}{
		"state":      1,
		"invoice_id": ctx.Query("titleId"),
		"apply_time": time.Now(),
	})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

// GetInvoiceByOpenid 对应 Java InvoiceController.getInvoiceByOpenid。
func (c *UserInvoiceController) GetInvoiceByOpenid(ctx *gin.Context) {
	var rows []model.Invoice
	conf.Db.Where("openid = ?", ctx.Query("openid")).Find(&rows)
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
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&t).Error == nil))
}

// GetByOpenid 对应 Java InvoiceController.getByOpenid：按当前登录用户 openid 查抬头。
func (c *UserInvoiceController) GetByOpenid(ctx *gin.Context) {
	var rows []model.InvoiceTitle
	conf.Db.Where("openid = ?", ctx.GetString("openid")).Find(&rows)
	list := make([]invoiceTitleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, invoiceTitleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetById 对应 Java InvoiceController.getById。
func (c *UserInvoiceController) GetById(ctx *gin.Context) {
	var t model.InvoiceTitle
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&t).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(invoiceTitleToDTO(t)))
}

// DeleteById 对应 Java InvoiceController.deleteById：先置发票 state=0 再删抬头。
func (c *UserInvoiceController) DeleteById(ctx *gin.Context) {
	id := ctx.Query("id")
	conf.Db.Model(&model.Invoice{}).Where("invoice_id = ?", id).Update("state", 0)
	res := conf.Db.Where("id = ?", id).Delete(&model.InvoiceTitle{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

// UpdetaById 对应 Java InvoiceController.updetaById：按 id 编辑发票抬头。
func (c *UserInvoiceController) UpdetaById(ctx *gin.Context) {
	var req invoiceTitleDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	res := conf.Db.Model(&model.InvoiceTitle{}).Where("id = ?", req.ID).Updates(invoiceTitleUpdates(req))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected))
}

// AddInvoice 对应 Java InvoiceController.addInvoice：实际为按 id 更新发票抬头。
func (c *UserInvoiceController) AddInvoice(ctx *gin.Context) {
	var req invoiceTitleDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	res := conf.Db.Model(&model.InvoiceTitle{}).Where("id = ?", req.ID).Updates(invoiceTitleUpdates(req))
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

// ============ PurchasePoleApplicationController（/purchasePoleApplication） ============

type PurchasePoleApplicationController struct{}

// UpdateOrderStatus 对应 Java PurchasePoleApplicationController.updateOrderStatus。
func (c *PurchasePoleApplicationController) UpdateOrderStatus(ctx *gin.Context) {
	res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
		Where("order_number = ?", ctx.Query("orderNumber")).
		Update("order_status", ctx.Query("orderStatus"))
	if res.RowsAffected == 0 {
		ctx.JSON(http.StatusOK, ResultError(400, "订单状态更新失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// GetByOrderStatus 对应 Java PurchasePoleApplicationController.getByOrderStatus（按申请时间倒序）。
func (c *PurchasePoleApplicationController) GetByOrderStatus(ctx *gin.Context) {
	var rows []model.PurchasePoleApplicationTbl
	conf.Db.Where("order_status = ?", ctx.Param("orderStatus")).Order("apply_time desc").Find(&rows)
	list := make([]purchasePoleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, purchasePoleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// Create 对应 Java PurchasePoleApplicationController.createPurchasePoleApplication。
func (c *PurchasePoleApplicationController) Create(ctx *gin.Context) {
	var req struct {
		Openid               string          `json:"openid"`
		Name                 string          `json:"name"`
		IDNumber             string          `json:"idNumber"`
		SelectedPackage      string          `json:"selectedPackage"`
		SelectedColor        string          `json:"selectedColor"`
		InstallationDistance float64         `json:"installationDistance"`
		UsageScenario        string          `json:"usageScenario"`
		InstallationAddress  string          `json:"installationAddress"`
		DetailedAddress      string          `json:"detailedAddress"`
		PhoneNumber          string          `json:"phoneNumber"`
		ActualPayment        decimal.Decimal `json:"actualPayment"`
		InstallationDate     string          `json:"installationDate"`
		PaymentType          string          `json:"paymentType"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "购桩申请创建异常: "+err.Error()))
		return
	}

	// 实付款为 0 → 待发货，否则待支付。
	orderStatus := "待支付"
	if req.ActualPayment.IsZero() {
		orderStatus = "待发货"
	}
	usageScenario := req.UsageScenario
	if strings.TrimSpace(usageScenario) == "" {
		usageScenario = "未指定"
	}

	p := model.PurchasePoleApplicationTbl{
		OrderNumber:          generateOrderNumber(),
		Openid:               req.Openid,
		Name:                 req.Name,
		IDNumber:             req.IDNumber,
		SelectedPackage:      req.SelectedPackage,
		SelectedColor:        req.SelectedColor,
		InstallationDistance: req.InstallationDistance,
		UsageScenario:        usageScenario,
		InstallationAddress:  req.InstallationAddress,
		DetailedAddress:      req.DetailedAddress,
		PhoneNumber:          req.PhoneNumber,
		ActualPayment:        req.ActualPayment,
		InstallationDate:     req.InstallationDate,
		PaymentType:          req.PaymentType,
		OrderStatus:          orderStatus,
		ApplyTime:            time.Now(),
	}
	if err := conf.Db.Create(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "购桩申请创建失败"))
		return
	}

	// 返回最新创建的订单号（按申请时间倒序第一条）。
	var latest model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("openid = ?", req.Openid).Order("apply_time desc").First(&latest).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "订单创建成功但未找到订单信息"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(latest.OrderNumber))
}

// generateOrderNumber 生成订单号：PPA + 年月日 + 6 位随机数（对齐 Java generateOrderNumber）。
func generateOrderNumber() string {
	return fmt.Sprintf("PPA%s%d", time.Now().Format("20060102"), rand.IntN(900000)+100000)
}

// GetCurrentUserApplication 对应 Java getCurrentUserApplication：取当前登录用户。
func (c *PurchasePoleApplicationController) GetCurrentUserApplication(ctx *gin.Context) {
	openid := ctx.GetString("openid")
	var rows []model.PurchasePoleApplicationTbl
	conf.Db.Where("openid = ?", openid).Order("apply_time desc").Find(&rows)
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultError(500, "未找到购桩申请信息"))
		return
	}
	list := make([]purchasePoleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, purchasePoleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetByOpenid 对应 Java getByOpenid（路径参数 openid，按申请时间倒序）。
func (c *PurchasePoleApplicationController) GetByOpenid(ctx *gin.Context) {
	openid := ctx.Param("openid")
	if openid == "" {
		openid = ctx.GetString("openid")
	}
	var rows []model.PurchasePoleApplicationTbl
	conf.Db.Where("openid = ?", openid).Order("apply_time desc").Find(&rows)
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultError(500, "未找到购桩申请信息"))
		return
	}
	list := make([]purchasePoleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, purchasePoleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// DeleteByOrderNumber 对应 Java deleteByOrderNumber。
func (c *PurchasePoleApplicationController) DeleteByOrderNumber(ctx *gin.Context) {
	res := conf.Db.Where("order_number = ?", ctx.Param("orderNumber")).Delete(&model.PurchasePoleApplicationTbl{})
	if res.RowsAffected == 0 {
		ctx.JSON(http.StatusOK, ResultError(400, "购桩申请删除失败，订单可能不存在"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// Pay 对应 Java pay：校验订单存在且为「待支付」，调用微信支付创建订单。
func (c *PurchasePoleApplicationController) Pay(ctx *gin.Context) {
	orderNumber := ctx.Param("orderNumber")
	var p model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", orderNumber).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "订单不存在"))
		return
	}
	if p.OrderStatus != "待支付" {
		ctx.JSON(http.StatusOK, ResultError(400, "订单状态不正确，无法支付"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(createPayScoreOrder(orderNumber, false, appid)))
}

// UpdatePaymentInfo 对应 Java updatePaymentInfo：更新交易号、支付时间并置状态为已支付。
func (c *PurchasePoleApplicationController) UpdatePaymentInfo(ctx *gin.Context) {
	payTime := time.Now()
	if v := ctx.Query("payTime"); v != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.Local); err == nil {
			payTime = t
		}
	}
	res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
		Where("order_number = ?", ctx.Query("orderNumber")).
		Updates(map[string]interface{}{
			"transaction_id": ctx.Query("transactionId"),
			"pay_time":       payTime,
			"order_status":   "已支付",
		})
	if res.RowsAffected == 0 {
		ctx.JSON(http.StatusOK, ResultError(400, "支付信息更新失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// GetByOrderNumber 对应 Java getByOrderNumber。
func (c *PurchasePoleApplicationController) GetByOrderNumber(ctx *gin.Context) {
	var p model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", ctx.Param("orderNumber")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "未找到购桩申请信息"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(purchasePoleToDTO(p)))
}

// ApplyAndProcessRefund 对应 Java applyAndProcessRefund：一次性完成申请与退款。
func (c *PurchasePoleApplicationController) ApplyAndProcessRefund(ctx *gin.Context) {
	var req struct {
		OrderNumber  string `json:"orderNumber"`
		RefundReason string `json:"refundReason"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: "+err.Error()))
		return
	}

	// 1. 验证订单是否存在
	var app model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", req.OrderNumber).First(&app).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 购桩订单不存在"))
		return
	}

	// 2. 验证订单状态是否支持退款
	if app.OrderStatus != "已支付" {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 订单状态不支持退款，当前状态："+app.OrderStatus))
		return
	}

	// 3. 验证支付信息是否完整
	if strings.TrimSpace(app.TransactionID) == "" {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 订单支付信息不完整，无法进行退款"))
		return
	}

	// 4. 生成退款单号
	refundOrderID := "REFUND_PPA_" + fmt.Sprintf("%d", time.Now().UnixMilli())

	// 5. 金额（元 → 分）
	refundAmount := int(app.ActualPayment.Mul(decimal.NewFromInt(100)).IntPart())
	totalAmount := refundAmount
	refundStatus, err := refundWxOrder(app.TransactionID, refundOrderID, refundAmount, totalAmount)
	if err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 微信退款失败: "+err.Error()))
		return
	}

	// 6. 处理退款结果
	switch refundStatus {
	case "SUCCESS":
		conf.Db.Model(&model.PurchasePoleApplicationTbl{}).Where("order_number = ?", req.OrderNumber).Updates(map[string]interface{}{
			"order_status":  "已退款",
			"refund_reason": req.RefundReason,
			"refund_time":   time.Now(),
		})
		ctx.JSON(http.StatusOK, ResultSuccess("退款成功"))
	case "PROCESSING", "CHANGE":
		conf.Db.Model(&model.PurchasePoleApplicationTbl{}).Where("order_number = ?", req.OrderNumber).Updates(map[string]interface{}{
			"order_status":  "退款中",
			"refund_reason": req.RefundReason,
			"refund_time":   time.Now(),
		})
		ctx.JSON(http.StatusOK, ResultSuccess("退款处理中"))
	default:
		conf.Db.Model(&model.PurchasePoleApplicationTbl{}).Where("order_number = ?", req.OrderNumber).Updates(map[string]interface{}{
			"order_status":  "退款失败",
			"refund_reason": req.RefundReason,
		})
		ctx.JSON(http.StatusOK, ResultError(500, "申请并处理退款异常: 退款失败，微信返回状态："+refundStatus))
	}
}
