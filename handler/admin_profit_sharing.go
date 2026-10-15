package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

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
// 对齐 Java ProfitSharingConfigSerImpl.getList：spacesId 解析为整数等值匹配，
// receiverMchId 模糊匹配，按 create_time 升序。
func (a *AdminProfitSharingController) List(c *gin.Context) {
	current, size := parsePage(c)
	total, rows := repository.ListProfitSharingConfigs(c.Query("spacesId"), c.Query("receiverMchId"), pageOffset(current, size), size)

	list := make([]profitSharingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, profitSharingToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// Detail GET /system/profitSharingConfig/detail/:id
func (a *AdminProfitSharingController) Detail(c *gin.Context) {
	var r model.TabProfitSharingConfig
	if err := repository.FindByID(&r, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(profitSharingToDTO(r)))
}

// Add POST /system/profitSharingConfig/add
// 对齐 Java ProfitSharingConfigSerImpl.addConfig：显式写入 create_time/update_time。
func (a *AdminProfitSharingController) Add(c *gin.Context) {
	var r model.TabProfitSharingConfig
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "添加分账配置失败"))
		return
	}
	now := time.Now()
	r.CreateTime = timePtr(now)
	r.UpdateTime = timePtr(now)
	if err := repository.Create(&r).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(500, "添加分账配置失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// Update PUT /system/profitSharingConfig/update
// 对齐 Java ProfitSharingConfigSerImpl.updateConfig：更新业务字段并刷新 update_time。
func (a *AdminProfitSharingController) Update(c *gin.Context) {
	var r model.TabProfitSharingConfig
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "修改分账配置失败"))
		return
	}
	res := repository.UpdateByID(&model.TabProfitSharingConfig{}, r.ID, map[string]interface{}{
		"spaces_id":       r.SpacesID,
		"receiver_mch_id": r.ReceiverMchID,
		"receiver_name":   r.ReceiverName,
		"sharing_ratio":   r.SharingRatio,
		"description":     r.Description,
		"enabled":         r.Enabled,
		"mode":            r.Mode,
		"update_time":     time.Now(),
	})
	if res.Error != nil {
		c.JSON(http.StatusOK, ResultError(500, "修改分账配置失败: "+res.Error.Error()))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// Delete DELETE /system/profitSharingConfig/delete/:id
func (a *AdminProfitSharingController) Delete(c *gin.Context) {
	repository.DeleteByID(&model.TabProfitSharingConfig{}, c.Param("id"))
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
		repository.DeleteProfitSharingConfigsByIDs(ids)
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}
