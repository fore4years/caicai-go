package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/model"
	"caicai-go/repository"
)

// privateChargingDTO 对应 Java domain.PrivateChargingBean（驼峰）。
type privateChargingDTO struct {
	CreateTime         *LocalDateTime  `json:"createTime,omitempty"`
	UpdateTime         *LocalDateTime  `json:"updateTime,omitempty"`
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
	ShareTime          string          `json:"shareTime,omitempty"`
	Fee                decimal.Decimal `json:"fee,omitempty"`
	ID                 int32           `json:"id"`
	ImageID            int32           `json:"imageId,omitempty"`
	Status             int32           `json:"status,omitempty"`
	OneClickOpening    int32           `json:"oneClickOpening,omitempty"`
	LedID              int32           `json:"ledId,omitempty"`
	HasCpLine          int32           `json:"hasCpLine,omitempty"`
	ExchangePlace      bool            `json:"exchangePlace,omitempty"`
	OfficialUse        bool            `json:"officialUse,omitempty"`
	OvertimeFee        bool            `json:"overtimeFee,omitempty"`
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
	repository.Count(&model.PrivateChargingTbl{}, &total)
	var rows []model.PrivateChargingTbl
	repository.Page(&model.PrivateChargingTbl{}, "create_time desc", pageOffset(current, size), size, &rows)
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
	total := repository.CountPrivateChargingByPhone(phone)
	rows := repository.PagePrivateChargingByPhone(phone, pageOffset(current, size), size)
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
	rowsAffected := repository.UpdatePrivateChargingStatus(req.ID, req.Status)
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
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
	rowsAffected := repository.UpdatePrivateChargingStatusByIDs(req.IDs, req.Status)
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}
