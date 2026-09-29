package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// PlaceApplyDTO 对应 Java domain.PlaceApplyBean，JSON 字段名对齐 Jackson 序列化（驼峰，null 省略）。
type PlaceApplyDTO struct {
	ID              string `json:"id,omitempty"`
	Ownerid         string `json:"ownerid,omitempty"`
	Name            string `json:"name,omitempty"`
	IDNumber        string `json:"idNumber,omitempty"`
	IDCardP         string `json:"idCard_p,omitempty"`
	IDCardR         string `json:"idCard_r,omitempty"`
	Prove           string `json:"prove,omitempty"`
	Province        string `json:"province,omitempty"`
	City            string `json:"city,omitempty"`
	District        string `json:"district,omitempty"`
	Code            string `json:"code,omitempty"`
	Streer          string `json:"streer,omitempty"`
	RelatedBuilding string `json:"relatedBuilding,omitempty"`
	Carnumber       string `json:"carnumber,omitempty"`
	ApplyTime       string `json:"applyTime,omitempty"`
	State           string `json:"state,omitempty"`
}

// placeApplyToDTO 将表模型映射为与 Java PlaceApplyBean 序列化一致的 DTO。
func placeApplyToDTO(m model.PlaceapplyTbl) PlaceApplyDTO {
	return PlaceApplyDTO{
		ID:              m.ID,
		Ownerid:         m.Ownerid,
		Name:            m.Name,
		IDNumber:        m.IDNumber,
		IDCardP:         m.IDCardP,
		IDCardR:         m.IDCardR,
		Prove:           m.Prove,
		Province:        m.Province,
		City:            m.City,
		District:        m.District,
		Code:            m.Code,
		Streer:          m.Streer,
		RelatedBuilding: m.RelatedBuilding,
		Carnumber:       m.Carnumber,
		ApplyTime:       m.ApplyTime,
		State:           m.State,
	}
}

// placeApplyList 将表模型切片映射为 DTO 切片。
func placeApplyList(rows []model.PlaceapplyTbl) []PlaceApplyDTO {
	list := make([]PlaceApplyDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, placeApplyToDTO(r))
	}
	return list
}

// AdminPlaceApplyController 对齐 Java System.controller.placeApplyCon（/system 车位申请审核）。
// 该控制器为遗留风格，返回原始 Integer/List/对象/String/Map（不包 Result）。
type AdminPlaceApplyController struct{}

// GetPlaceApplyMax GET /system/getPlaceApplyMax → 总数（原始 Integer）
func (a *AdminPlaceApplyController) GetPlaceApplyMax(c *gin.Context) {
	var count int64
	conf.Db.Model(&model.PlaceapplyTbl{}).Count(&count)
	c.JSON(http.StatusOK, count)
}

// GetPlaceApplyPaging GET /system/getPlaceApplyPaging?page=&length= → 分页列表（原始 List）
func (a *AdminPlaceApplyController) GetPlaceApplyPaging(c *gin.Context) {
	current, size := parsePage(c)
	var rows []model.PlaceapplyTbl
	conf.Db.Model(&model.PlaceapplyTbl{}).
		Order("applyTime desc").
		Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	c.JSON(http.StatusOK, placeApplyList(rows))
}

// GetPlaceApplyById GET /system/getPlaceApplyById?id= → 单条（原始对象，未找到返回 null）
func (a *AdminPlaceApplyController) GetPlaceApplyById(c *gin.Context) {
	var pa model.PlaceapplyTbl
	if err := conf.Db.Where("id = ?", c.Query("id")).First(&pa).Error; err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, placeApplyToDTO(pa))
}

// DoPlaceApply POST /system/doPlaceApply body {state,id} → 审核结果消息（原始 String）
// 忠实还原 Java placeApplySerImpl.doPlaceApply：
//   - state=="no"：placeapply 置“未通过”，写 message_tbl（status=未读）
//   - 其他：已“已通过”则跳过；否则置“已通过”，生成 placeid 写入 place_tbl，写 message_tbl（status 为空）
func (a *AdminPlaceApplyController) DoPlaceApply(c *gin.Context) {
	var req struct {
		State string `json:"state"`
		ID    string `json:"id"`
	}
	_ = c.ShouldBindJSON(&req)

	msg := "已通过车位申请"
	now := sysTime()

	if req.State == "no" {
		// 驳回
		conf.Db.Model(&model.PlaceapplyTbl{}).Where("id = ?", req.ID).Update("state", "未通过")
		msg = "已拒绝车位申请"
		var byID model.PlaceapplyTbl
		if err := conf.Db.Where("id = ?", req.ID).First(&byID).Error; err == nil {
			text := byID.Name + "先生（女士），你好，你注册的车位：" + byID.Province + byID.City + byID.Streer + byID.RelatedBuilding + ",现在暂未通过，了解详情，请咨询工作人员~"
			conf.Db.Exec("INSERT INTO message_tbl (ownerid, msgtime, message, status) VALUES (?,?,?,?)",
				byID.Ownerid, now, text, "未读")
		}
	} else {
		var pa model.PlaceapplyTbl
		if err := conf.Db.Where("id = ?", req.ID).First(&pa).Error; err != nil {
			c.String(http.StatusOK, msg)
			return
		}
		if pa.State == "已通过" {
			// 已通过，不做任何变更
			c.String(http.StatusOK, msg)
			return
		}

		// 通过
		conf.Db.Model(&model.PlaceapplyTbl{}).Where("id = ?", req.ID).Update("state", "已通过")
		var byID model.PlaceapplyTbl
		conf.Db.Where("id = ?", req.ID).First(&byID)

		// 生成 placeid：code + 5 位递增序号（对齐 Java selectMaxID + shuzi.tianchongling）
		placeid := ""
		var p model.PlaceTbl
		if err := conf.Db.Where("placeid LIKE ?", pa.Code+"%").Order("placeid desc").Limit(1).First(&p).Error; err != nil {
			placeid = pa.Code + "00001"
		} else {
			suffix := strings.TrimPrefix(p.Placeid, pa.Code)
			n := 0
			if v, e := strconv.Atoi(suffix); e == nil {
				n = v
			}
			placeid = pa.Code + fmt.Sprintf("%05d", n+1)
		}

		// 写入 place_tbl（对齐 Java zhucecheweiInsert 的 12 个字段，state=待加锁）
		in := model.PlaceTbl{
			Placeid:         placeid,
			Ownerid:         pa.Ownerid,
			Province:        pa.Province,
			City:            pa.City,
			District:        pa.District,
			Streer:          pa.Streer,
			RelatedBuilding: pa.RelatedBuilding,
			State:           "待加锁",
		}
		conf.Db.Select(
			"placeid", "ownerid", "province", "city", "district", "streer", "relatedBuilding",
			"longitude", "latitude", "rate", "state", "fixType",
		).Create(&in)

		text := byID.Name + "先生（女士），您好，您注册的车位：" + byID.Province + byID.City + byID.Streer + byID.RelatedBuilding + ",现已通过，您接下来可以添加车位锁以共享您的车位啦~"
		conf.Db.Exec("INSERT INTO message_tbl (ownerid, msgtime, message, status) VALUES (?,?,?,?)",
			pa.Ownerid, now, text, nil)
	}
	c.String(http.StatusOK, msg)
}

// PaChange GET /system/paChange?tiaojian=&page=&length= → {max,list}（按 state 过滤）
func (a *AdminPlaceApplyController) PaChange(c *gin.Context) {
	tiaojian := c.Query("tiaojian")
	current, size := parsePage(c)

	var max int64
	conf.Db.Model(&model.PlaceapplyTbl{}).Where("state = ?", tiaojian).Count(&max)

	var rows []model.PlaceapplyTbl
	conf.Db.Model(&model.PlaceapplyTbl{}).Where("state = ?", tiaojian).
		Order("applyTime desc").Offset(pageOffset(current, size)).Limit(size).Find(&rows)

	c.JSON(http.StatusOK, gin.H{"max": max, "list": placeApplyList(rows)})
}

// MushDel POST /system/mushDel body {xuanzhong:[id...]} → "删除成功！"
func (a *AdminPlaceApplyController) MushDel(c *gin.Context) {
	var req struct {
		Xuanzhong []string `json:"xuanzhong"`
	}
	_ = c.ShouldBindJSON(&req)
	for _, id := range req.Xuanzhong {
		conf.Db.Where("id = ?", id).Delete(&model.PlaceapplyTbl{})
	}
	c.String(http.StatusOK, "删除成功！")
}

// Sousuo GET /system/sousuo?tiaojian=&page=&length= → {max,list} 或 null
// 依次在 applyTime/name/idNumber/province/city/district/streer 中做 LIKE 匹配，取首个命中的字段。
func (a *AdminPlaceApplyController) Sousuo(c *gin.Context) {
	tiaojian := c.Query("tiaojian")
	current, size := parsePage(c)

	fields := []string{"applyTime", "name", "idNumber", "province", "city", "district", "streer"}
	for _, f := range fields {
		var max int64
		conf.Db.Model(&model.PlaceapplyTbl{}).Where(f+" LIKE ?", "%"+tiaojian+"%").Count(&max)
		if max != 0 {
			var rows []model.PlaceapplyTbl
			conf.Db.Model(&model.PlaceapplyTbl{}).Where(f+" LIKE ?", "%"+tiaojian+"%").
				Order("applyTime desc").Offset(pageOffset(current, size)).Limit(size).Find(&rows)
			c.JSON(http.StatusOK, gin.H{"max": max, "list": placeApplyList(rows)})
			return
		}
	}
	c.JSON(http.StatusOK, nil)
}

// RegisterAdminPlaceApply 注册 /system 下车位申请审核相关路由（对齐 Java placeApplyCon）。
func RegisterAdminPlaceApply(r *gin.Engine) {
	c := new(AdminPlaceApplyController)
	r.Any("/system/getPlaceApplyMax", c.GetPlaceApplyMax)
	r.Any("/system/getPlaceApplyPaging", c.GetPlaceApplyPaging)
	r.Any("/system/getPlaceApplyById", c.GetPlaceApplyById)
	r.Any("/system/doPlaceApply", c.DoPlaceApply)
	r.Any("/system/paChange", c.PaChange)
	r.Any("/system/mushDel", c.MushDel)
	r.Any("/system/sousuo", c.Sousuo)
}
