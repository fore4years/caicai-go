package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// invoiceDTO 对应 Java domain.Invoice（invoice，驼峰）。
type invoiceDTO struct {
	ID          int32          `json:"id"`
	InvoiceType string         `json:"invoiceType,omitempty"`
	InvoiceDate *LocalDateTime `json:"invoiceDate,omitempty"`
	FarmTax     string         `json:"farmTax,omitempty"`
	FarmName    string         `json:"farmName,omitempty"`
	FarmAddress string         `json:"farmAddress,omitempty"`
	InvoiceCode string         `json:"invoiceCode,omitempty"`
	InvoiceNum  string         `json:"invoiceNum,omitempty"`
	Amount      float64        `json:"amount,omitempty"`
	TaxAmount   float64        `json:"taxAmount,omitempty"`
	InvoiceID   int32          `json:"invoiceId,omitempty"`
	State       int32          `json:"state,omitempty"`
	Openid      string         `json:"openid,omitempty"`
	ApplyTime   *LocalDateTime `json:"applyTime,omitempty"`
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

// AdminInvoiceController 对齐 Java System.controller.InvoiceDetailCon。
type AdminInvoiceController struct{}

// List GET /system/invoice/list
func (a *AdminInvoiceController) List(c *gin.Context) {
	current, size := parsePage(c)
	openid := c.Query("openid")
	phone := c.Query("phone")

	query := func() *gorm.DB {
		db := conf.Db.Model(&model.Invoice{})
		if openid != "" {
			db = db.Where("openid = ?", openid)
		}
		if phone != "" {
			var u model.UserTbl
			if err := conf.Db.Where("phone = ?", phone).First(&u).Error; err == nil {
				db = db.Where("openid = ?", u.Openid)
			}
		}
		return db
	}

	var total int64
	query().Count(&total)
	var rows []model.Invoice
	query().Order("apply_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&rows)

	list := make([]invoiceDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, invoiceToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(gin.H{"total": total, "list": list}))
}

// Process PUT /system/invoice/process?id=
func (a *AdminInvoiceController) Process(c *gin.Context) {
	res := conf.Db.Model(&model.Invoice{}).Where("id = ? AND state = 1", c.Query("id")).Update("state", 2)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusOK, ResultError(500, "发票处理失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("发票处理成功"))
}

// BatchDelete POST /system/invoice/batchDelete
func (a *AdminInvoiceController) BatchDelete(c *gin.Context) {
	var ids []int32
	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
		return
	}
	if len(ids) > 0 {
		conf.Db.Where("id IN ?", ids).Delete(&model.Invoice{})
	}
	c.JSON(http.StatusOK, ResultSuccess("批量删除成功"))
}

// CheckRecentApply GET /system/invoice/checkRecentApply
func (a *AdminInvoiceController) CheckRecentApply(c *gin.Context) {
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	var n int64
	conf.Db.Model(&model.Invoice{}).Where("state = 1 AND apply_time >= ?", twoDaysAgo).Count(&n)
	c.JSON(http.StatusOK, ResultSuccess(n > 0))
}

// AdminBalanceController 对齐 Java System.controller.BalanceDetailCon。
type AdminBalanceController struct{}

// List GET /system/balanceDetail/list
func (a *AdminBalanceController) List(c *gin.Context) {
	current, size := parsePage(c)
	openid := c.Query("openid")
	phone := c.Query("phone")

	query := func() *gorm.DB {
		db := conf.Db.Model(&model.UserTbl{})
		if openid != "" {
			db = db.Where("openid = ?", openid)
		}
		if phone != "" {
			db = db.Where("phone LIKE ?", "%"+phone+"%")
		}
		return db
	}

	var total int64
	query().Count(&total)
	var users []model.UserTbl
	query().Offset(pageOffset(current, size)).Limit(size).Find(&users)

	list := make([]UserDTO, 0, len(users))
	for _, u := range users {
		list = append(list, userToDTO(u))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// BalanceSource GET /system/balanceDetail/balanceSource?openid=
func (a *AdminBalanceController) BalanceSource(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(http.StatusOK, ResultError(400, "用户openid不能为空"))
		return
	}
	var u model.UserTbl
	if err := conf.Db.Where("openid = ?", openid).First(&u).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(500, "获取余额来源失败: 用户不存在"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"balans":       u.Balans,
		"freezeBalans": u.FreezeBalans,
	}))
}

// Recharge POST /system/balanceDetail/recharge?openid=&amount=
func (a *AdminBalanceController) Recharge(c *gin.Context) {
	openid := c.Query("openid")
	amount := queryFloat(c, "amount")
	if openid == "" {
		c.JSON(http.StatusOK, ResultError(400, "用户openid不能为空"))
		return
	}
	if amount <= 0 {
		c.JSON(http.StatusOK, ResultError(400, "充值金额必须大于0"))
		return
	}
	conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).
		UpdateColumn("balans", gorm.Expr("balans + ?", amount))
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// Deduct POST /system/balanceDetail/deduct?openid=&amount=
func (a *AdminBalanceController) Deduct(c *gin.Context) {
	openid := c.Query("openid")
	amount := queryFloat(c, "amount")
	if openid == "" {
		c.JSON(http.StatusOK, ResultError(400, "用户openid不能为空"))
		return
	}
	if amount <= 0 {
		c.JSON(http.StatusOK, ResultError(400, "扣除金额必须大于0"))
		return
	}
	conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).
		UpdateColumn("balans", gorm.Expr("balans - ?", amount))
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// hotelDTO 对应 Java domain.Hotel（hotel_tbl，驼峰）。
type hotelDTO struct {
	ID            int32          `json:"id"`
	Openid        string         `json:"openid,omitempty"`
	HotelName     string         `json:"hotelName,omitempty"`
	HotelAddress  string         `json:"hotelAddress,omitempty"`
	ParkingNumber int32          `json:"parkingNumber,omitempty"`
	Phone         string         `json:"phone,omitempty"`
	Email         string         `json:"email,omitempty"`
	HotelTime     string         `json:"hotelTime,omitempty"`
	HotelImgID    int32          `json:"hotelImgId,omitempty"`
	Review        int32          `json:"review,omitempty"`
	CreatTime     *LocalDateTime `json:"creatTime,omitempty"`
}

func hotelToDTO(h model.HotelTbl) hotelDTO {
	return hotelDTO{
		ID:            h.ID,
		Openid:        h.Openid,
		HotelName:     h.HotelName,
		HotelAddress:  h.HotelAddress,
		ParkingNumber: h.ParkingNumber,
		Phone:         h.Phone,
		Email:         h.Email,
		HotelTime:     h.HotelTime,
		HotelImgID:    h.HotelImgID,
		Review:        h.Review,
		CreatTime:     timeToLocal(h.CreatTime),
	}
}

// AdminHotelController 对齐 Java System.controller.HotelCon。
type AdminHotelController struct{}

// List GET /system/hotel/list
func (a *AdminHotelController) List(c *gin.Context) {
	current, size := parsePage(c)
	id := c.Query("id")
	openid := c.Query("openid")
	hotelName := c.Query("hotelName")

	query := func() *gorm.DB {
		db := conf.Db.Model(&model.HotelTbl{})
		if id != "" {
			db = db.Where("id = ?", id)
		}
		if openid != "" {
			db = db.Where("openid = ?", openid)
		}
		if hotelName != "" {
			db = db.Where("hotel_name LIKE ?", "%"+hotelName+"%")
		}
		return db
	}

	var total int64
	query().Count(&total)
	var rows []model.HotelTbl
	query().Order("creat_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&rows)

	list := make([]hotelDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, hotelToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(gin.H{"total": total, "list": list}))
}

// UpdateReview PUT /system/hotel/updateReview?id=&review=
func (a *AdminHotelController) UpdateReview(c *gin.Context) {
	res := conf.Db.Model(&model.HotelTbl{}).Where("id = ?", c.Query("id")).Update("review", c.Query("review"))
	if res.RowsAffected == 0 {
		c.JSON(http.StatusOK, ResultError(500, "更新审核状态失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("审核状态更新成功"))
}

// BatchDelete POST /system/hotel/batchDelete
func (a *AdminHotelController) BatchDelete(c *gin.Context) {
	var ids []int32
	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
		return
	}
	if len(ids) > 0 {
		conf.Db.Where("id IN ?", ids).Delete(&model.HotelTbl{})
	}
	c.JSON(http.StatusOK, ResultSuccess("批量删除成功"))
}

// profitSharingDTO 对应 Java model.ProfitSharingConfigBean（驼峰）。
type profitSharingDTO struct {
	ID            int32          `json:"id"`
	SpacesID      int32          `json:"spacesId,omitempty"`
	ReceiverMchID string         `json:"receiverMchId,omitempty"`
	ReceiverName  string         `json:"receiverName,omitempty"`
	SharingRatio  int32          `json:"sharingRatio,omitempty"`
	Description   string         `json:"description,omitempty"`
	Enabled       bool           `json:"enabled,omitempty"`
	Mode          int32          `json:"mode,omitempty"`
	CreateTime    *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime    *LocalDateTime `json:"updateTime,omitempty"`
}

func profitSharingToDTO(p model.TabProfitSharingConfig) profitSharingDTO {
	return profitSharingDTO{
		ID:            p.ID,
		SpacesID:      p.SpacesID,
		ReceiverMchID: p.ReceiverMchID,
		ReceiverName:  p.ReceiverName,
		SharingRatio:  p.SharingRatio,
		Description:   p.Description,
		Enabled:       p.Enabled,
		Mode:          p.Mode,
		CreateTime:    timeToLocal(p.CreateTime),
		UpdateTime:    timeToLocal(p.UpdateTime),
	}
}

// AdminProfitSharingController 对齐 Java System.controller.ProfitSharingConfigCon。
type AdminProfitSharingController struct{}

// List GET /system/profitSharingConfig/list
func (a *AdminProfitSharingController) List(c *gin.Context) {
	current, size := parsePage(c)
	spacesID := c.Query("spacesId")
	receiverMchID := c.Query("receiverMchId")

	query := func() *gorm.DB {
		db := conf.Db.Model(&model.TabProfitSharingConfig{})
		if spacesID != "" {
			db = db.Where("spaces_id = ?", spacesID)
		}
		if receiverMchID != "" {
			db = db.Where("receiver_mch_id = ?", receiverMchID)
		}
		return db
	}

	var total int64
	query().Count(&total)
	var rows []model.TabProfitSharingConfig
	query().Offset(pageOffset(current, size)).Limit(size).Find(&rows)

	list := make([]profitSharingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, profitSharingToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// Detail GET /system/profitSharingConfig/detail/:id
func (a *AdminProfitSharingController) Detail(c *gin.Context) {
	var r model.TabProfitSharingConfig
	if err := conf.Db.Where("id = ?", c.Param("id")).First(&r).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(profitSharingToDTO(r)))
}

// Add POST /system/profitSharingConfig/add
func (a *AdminProfitSharingController) Add(c *gin.Context) {
	var r model.TabProfitSharingConfig
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "添加分账配置失败"))
		return
	}
	if err := conf.Db.Create(&r).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(500, "添加分账配置失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// Update PUT /system/profitSharingConfig/update
func (a *AdminProfitSharingController) Update(c *gin.Context) {
	var r model.TabProfitSharingConfig
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "修改分账配置失败"))
		return
	}
	res := conf.Db.Model(&model.TabProfitSharingConfig{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
		"spaces_id":       r.SpacesID,
		"receiver_mch_id": r.ReceiverMchID,
		"receiver_name":   r.ReceiverName,
		"sharing_ratio":   r.SharingRatio,
		"description":     r.Description,
		"enabled":         r.Enabled,
		"mode":            r.Mode,
	})
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// Delete DELETE /system/profitSharingConfig/delete/:id
func (a *AdminProfitSharingController) Delete(c *gin.Context) {
	conf.Db.Where("id = ?", c.Param("id")).Delete(&model.TabProfitSharingConfig{})
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// BatchDelete DELETE /system/profitSharingConfig/batchDelete
func (a *AdminProfitSharingController) BatchDelete(c *gin.Context) {
	var ids []int32
	if err := c.ShouldBindJSON(&ids); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "批量删除分账配置失败"))
		return
	}
	if len(ids) > 0 {
		conf.Db.Where("id IN ?", ids).Delete(&model.TabProfitSharingConfig{})
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// purchasePoleDTO 对应 Java domain.PurchasePoleApplication（驼峰）。
type purchasePoleDTO struct {
	OrderNumber          string         `json:"orderNumber"`
	Openid               string         `json:"openid,omitempty"`
	Name                 string         `json:"name,omitempty"`
	IDNumber             string         `json:"idNumber,omitempty"`
	PhoneNumber          string         `json:"phoneNumber,omitempty"`
	SelectedPackage      string         `json:"selectedPackage,omitempty"`
	SelectedColor        string         `json:"selectedColor,omitempty"`
	InstallationDistance float64        `json:"installationDistance,omitempty"`
	InstallationDate     string         `json:"installationDate,omitempty"`
	UsageScenario        string         `json:"usageScenario,omitempty"`
	InstallationAddress  string         `json:"installationAddress,omitempty"`
	DetailedAddress      string         `json:"detailedAddress,omitempty"`
	ApplyTime            *LocalDateTime `json:"applyTime,omitempty"`
	OrderStatus          string         `json:"orderStatus,omitempty"`
	ActualPayment        float64        `json:"actualPayment,omitempty"`
	PayTime              *LocalDateTime `json:"payTime,omitempty"`
	TransactionID        string         `json:"transactionId,omitempty"`
	RefundReason         string         `json:"refundReason,omitempty"`
	RefundTime           *LocalDateTime `json:"refundTime,omitempty"`
	PaymentType          string         `json:"paymentType,omitempty"`
}

func purchasePoleToDTO(p model.PurchasePoleApplicationTbl) purchasePoleDTO {
	return purchasePoleDTO{
		OrderNumber:          p.OrderNumber,
		Openid:               p.Openid,
		Name:                 p.Name,
		IDNumber:             p.IDNumber,
		PhoneNumber:          p.PhoneNumber,
		SelectedPackage:      p.SelectedPackage,
		SelectedColor:        p.SelectedColor,
		InstallationDistance: p.InstallationDistance,
		InstallationDate:     p.InstallationDate,
		UsageScenario:        p.UsageScenario,
		InstallationAddress:  p.InstallationAddress,
		DetailedAddress:      p.DetailedAddress,
		ApplyTime:            timeToLocal(p.ApplyTime),
		OrderStatus:          p.OrderStatus,
		ActualPayment:        p.ActualPayment,
		PayTime:              timeToLocal(p.PayTime),
		TransactionID:        p.TransactionID,
		RefundReason:         p.RefundReason,
		RefundTime:           timeToLocal(p.RefundTime),
		PaymentType:          p.PaymentType,
	}
}

// AdminPurchasePoleController 对齐 Java System.controller.PurchasePoleManageCon。
type AdminPurchasePoleController struct{}

// List GET /system/purchasePoleManage/list
func (a *AdminPurchasePoleController) List(c *gin.Context) {
	current, size := parsePage(c)
	orderNumber := c.Query("orderNumber")
	openid := c.Query("openid")
	orderStatus := c.Query("orderStatus")

	query := func() *gorm.DB {
		db := conf.Db.Model(&model.PurchasePoleApplicationTbl{})
		if orderNumber != "" {
			db = db.Where("order_number = ?", orderNumber)
		}
		if openid != "" {
			db = db.Where("openid = ?", openid)
		}
		if orderStatus != "" {
			db = db.Where("order_status = ?", orderStatus)
		}
		return db
	}

	var total int64
	query().Count(&total)
	var rows []model.PurchasePoleApplicationTbl
	query().Order("apply_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&rows)

	list := make([]purchasePoleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, purchasePoleToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// Detail GET /system/purchasePoleManage/detail?orderNumber=
func (a *AdminPurchasePoleController) Detail(c *gin.Context) {
	var r model.PurchasePoleApplicationTbl
	if err := conf.Db.Where("order_number = ?", c.Query("orderNumber")).First(&r).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(400, "未找到购桩订单详情"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(purchasePoleToDTO(r)))
}

// UpdateStatus POST /system/purchasePoleManage/updateStatus
func (a *AdminPurchasePoleController) UpdateStatus(c *gin.Context) {
	res := conf.Db.Model(&model.PurchasePoleApplicationTbl{}).
		Where("order_number = ?", c.Query("orderNumber")).
		Update("order_status", c.Query("orderStatus"))
	if res.RowsAffected == 0 {
		c.JSON(http.StatusOK, ResultError(500, "更新购桩订单状态失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("订单状态更新成功"))
}

// BatchDelete POST /system/purchasePoleManage/batchDelete
func (a *AdminPurchasePoleController) BatchDelete(c *gin.Context) {
	var orderNumbers []string
	if err := c.ShouldBindJSON(&orderNumbers); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
		return
	}
	if len(orderNumbers) > 0 {
		conf.Db.Where("order_number IN ?", orderNumbers).Delete(&model.PurchasePoleApplicationTbl{})
	}
	c.JSON(http.StatusOK, ResultSuccess("批量删除成功"))
}

// privateChargingDTO 对应 Java domain.PrivateChargingBean（驼峰）。
type privateChargingDTO struct {
	ID                 int32          `json:"id"`
	Openid             string         `json:"openid,omitempty"`
	Name               string         `json:"name,omitempty"`
	IDCard             string         `json:"idCard,omitempty"`
	Phone              string         `json:"phone,omitempty"`
	CommunityName      string         `json:"communityName,omitempty"`
	CommunityAddress   string         `json:"communityAddress,omitempty"`
	CommunitySpacesNum string         `json:"communitySpacesNum,omitempty"`
	ProductType        string         `json:"productType,omitempty"`
	ElectricityType    string         `json:"electricityType,omitempty"`
	ProductID          string         `json:"productId,omitempty"`
	SpacesNum          string         `json:"spacesNum,omitempty"`
	ImageID            int32          `json:"imageId,omitempty"`
	Status             int32          `json:"status,omitempty"`
	OneClickOpening    int32          `json:"oneClickOpening,omitempty"`
	ShareTime          string         `json:"shareTime,omitempty"`
	ExchangePlace      bool           `json:"exchangePlace,omitempty"`
	Fee                float64        `json:"fee,omitempty"`
	OfficialUse        bool           `json:"officialUse,omitempty"`
	OvertimeFee        bool           `json:"overtimeFee,omitempty"`
	LedID              int32          `json:"ledId,omitempty"`
	HasCpLine          int32          `json:"hasCpLine,omitempty"`
	CreateTime         *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime         *LocalDateTime `json:"updateTime,omitempty"`
}

func privateChargingToDTO(p model.PrivateChargingTbl) privateChargingDTO {
	return privateChargingDTO{
		ID:                 p.ID,
		Openid:             p.Openid,
		Name:               p.Name,
		IDCard:             p.IDCard,
		Phone:              p.Phone,
		CommunityName:      p.CommunityName,
		CommunityAddress:   p.CommunityAddress,
		CommunitySpacesNum: p.CommunitySpacesNum,
		ProductType:        p.ProductType,
		ElectricityType:    p.ElectricityType,
		ProductID:          p.ProductID,
		SpacesNum:          p.SpacesNum,
		ImageID:            p.ImageID,
		Status:             p.Status,
		OneClickOpening:    p.OneClickOpening,
		ShareTime:          p.ShareTime,
		ExchangePlace:      p.ExchangePlace,
		Fee:                p.Fee,
		OfficialUse:        p.OfficialUse,
		OvertimeFee:        p.OvertimeFee,
		LedID:              p.LedID,
		HasCpLine:          p.HasCpLine,
		CreateTime:         timeToLocal(p.CreateTime),
		UpdateTime:         timeToLocal(p.UpdateTime),
	}
}

// AdminPrivatePileController 对齐 Java System.controller.PrivatePileAuditController。
type AdminPrivatePileController struct{}

// GetAuditList GET /system/privatePile/getAuditList
func (a *AdminPrivatePileController) GetAuditList(c *gin.Context) {
	current, size := parsePage(c)
	var total int64
	conf.Db.Model(&model.PrivateChargingTbl{}).Count(&total)
	var rows []model.PrivateChargingTbl
	conf.Db.Model(&model.PrivateChargingTbl{}).Order("create_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]privateChargingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateChargingToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// SearchByPhone GET /system/privatePile/searchByPhone
func (a *AdminPrivatePileController) SearchByPhone(c *gin.Context) {
	current, size := parsePage(c)
	phone := c.Query("phone")
	var total int64
	conf.Db.Model(&model.PrivateChargingTbl{}).Where("phone LIKE ?", "%"+phone+"%").Count(&total)
	var rows []model.PrivateChargingTbl
	conf.Db.Model(&model.PrivateChargingTbl{}).Where("phone LIKE ?", "%"+phone+"%").
		Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]privateChargingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateChargingToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// UpdateStatus POST /system/privatePile/updateStatus body {id,status}
func (a *AdminPrivatePileController) UpdateStatus(c *gin.Context) {
	var req struct {
		ID     int32 `json:"id"`
		Status int32 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id = ?", req.ID).Update("status", req.Status)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// BatchUpdateStatus POST /system/privatePile/batchUpdateStatus body {ids:[],status}
func (a *AdminPrivatePileController) BatchUpdateStatus(c *gin.Context) {
	var req struct {
		IDs    []int32 `json:"ids"`
		Status int32   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id IN ?", req.IDs).Update("status", req.Status)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}
