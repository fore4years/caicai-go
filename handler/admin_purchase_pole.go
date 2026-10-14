package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// purchasePoleDTO 对应 Java domain.PurchasePoleApplication（驼峰）。
type purchasePoleDTO struct {
	OrderNumber          string          `json:"orderNumber"`
	Openid               string          `json:"openid,omitempty"`
	Name                 string          `json:"name,omitempty"`
	IDNumber             string          `json:"idNumber,omitempty"`
	PhoneNumber          string          `json:"phoneNumber,omitempty"`
	SelectedPackage      string          `json:"selectedPackage,omitempty"`
	SelectedColor        string          `json:"selectedColor,omitempty"`
	InstallationDistance float64         `json:"installationDistance,omitempty"`
	InstallationDate     string          `json:"installationDate,omitempty"`
	UsageScenario        string          `json:"usageScenario,omitempty"`
	InstallationAddress  string          `json:"installationAddress,omitempty"`
	DetailedAddress      string          `json:"detailedAddress,omitempty"`
	ApplyTime            *LocalDateTime  `json:"applyTime,omitempty"`
	OrderStatus          string          `json:"orderStatus,omitempty"`
	ActualPayment        decimal.Decimal `json:"actualPayment,omitempty"`
	PayTime              *LocalDateTime  `json:"payTime,omitempty"`
	TransactionID        string          `json:"transactionId,omitempty"`
	RefundReason         string          `json:"refundReason,omitempty"`
	RefundTime           *LocalDateTime  `json:"refundTime,omitempty"`
	PaymentType          string          `json:"paymentType,omitempty"`
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
		InstallationDistance: p.InstallationDistance.InexactFloat64(),
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
			// 处理"退款中"状态识别为"已退款"
			if orderStatus == "退款中" {
				db = db.Where("order_status = ?", "已退款")
			} else {
				db = db.Where("order_status = ?", orderStatus)
			}
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
	if len(orderNumbers) == 0 {
		c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
		return
	}
	res := conf.Db.Where("order_number IN ?", orderNumbers).Delete(&model.PurchasePoleApplicationTbl{})
	if res.RowsAffected > 0 {
		c.JSON(http.StatusOK, ResultSuccess("批量删除成功"))
		return
	}
	c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
}
