package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// privateChargingDTO 对应 Java domain.PrivateChargingBean（驼峰）。
type privateChargingDTO struct {
	ID                 int32           `json:"id"`
	Openid             string          `json:"openid,omitempty"`
	Name               string          `json:"name,omitempty"`
	IDCard             string          `json:"idCard,omitempty"`
	Phone              string          `json:"phone,omitempty"`
	CommunityName      string          `json:"communityName,omitempty"`
	CommunityAddress   string          `json:"communityAddress,omitempty"`
	CommunitySpacesNum string          `json:"communitySpacesNum,omitempty"`
	ProductType        string          `json:"productType,omitempty"`
	ElectricityType    string          `json:"electricityType,omitempty"`
	ProductID          string          `json:"productId,omitempty"`
	SpacesNum          string          `json:"spacesNum,omitempty"`
	ImageID            int32           `json:"imageId,omitempty"`
	Status             int32           `json:"status,omitempty"`
	OneClickOpening    int32           `json:"oneClickOpening,omitempty"`
	ShareTime          string          `json:"shareTime,omitempty"`
	ExchangePlace      bool            `json:"exchangePlace,omitempty"`
	Fee                decimal.Decimal `json:"fee,omitempty"`
	OfficialUse        bool            `json:"officialUse,omitempty"`
	OvertimeFee        bool            `json:"overtimeFee,omitempty"`
	LedID              int32           `json:"ledId,omitempty"`
	HasCpLine          int32           `json:"hasCpLine,omitempty"`
	CreateTime         *LocalDateTime  `json:"createTime,omitempty"`
	UpdateTime         *LocalDateTime  `json:"updateTime,omitempty"`
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
