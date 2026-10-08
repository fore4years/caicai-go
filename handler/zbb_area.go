package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// zbbAreaDTO 对应 Java ZbbAreaBean，JSON 字段名对齐 Jackson 序列化（驼峰）。
type zbbAreaDTO struct {
	ID          int32   `json:"id,omitempty"`
	Pid         int32   `json:"pid,omitempty"`
	Code        string  `json:"code,omitempty"`
	Name        string  `json:"name,omitempty"`
	Fullname    string  `json:"fullname,omitempty"`
	Pinyin      string  `json:"pinyin,omitempty"`
	FirstLetter string  `json:"firstLetter,omitempty"`
	FullLetter  string  `json:"fullLetter,omitempty"`
	Lat         float64 `json:"lat,omitempty"`
	Lng         float64 `json:"lng,omitempty"`
	Level       int32   `json:"level,omitempty"`
}

// zbbAreaToDTO 将 model.ZbbArea 转为 zbbAreaDTO。
func zbbAreaToDTO(a model.ZbbArea) zbbAreaDTO {
	return zbbAreaDTO{
		ID:          a.ID,
		Pid:         a.Pid,
		Code:        a.Code,
		Name:        a.Name,
		Fullname:    a.Fullname,
		Pinyin:      a.Pinyin,
		FirstLetter: a.FirstLetter,
		FullLetter:  a.FullLetter,
		Lat:         a.Lat,
		Lng:         a.Lng,
		Level:       a.Level,
	}
}

// zbbAreaReq 对应 Java ZbbAreaBean 的请求体（驼峰字段名，与 model.ZbbArea 的蛇形 json tag 不同）。
type zbbAreaReq struct {
	Pid         int32   `json:"pid"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Fullname    string  `json:"fullname"`
	Pinyin      string  `json:"pinyin"`
	FirstLetter string  `json:"firstLetter"`
	FullLetter  string  `json:"fullLetter"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Level       int32   `json:"level"`
}

// ============ ZbbAreaController（/ZbbAreaBean，表 zbb_area） ============

type ZbbAreaController struct{}

// GetById GET /ZbbAreaBean/getById 对齐 Java getById：按主键查 zbb_area。
func (c *ZbbAreaController) GetById(ctx *gin.Context) {
	var a model.ZbbArea
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&a).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(zbbAreaToDTO(a)))
}

// Add POST /ZbbAreaBean/add 对齐 Java add：insert 一条 zbb_area。
func (c *ZbbAreaController) Add(ctx *gin.Context) {
	var req zbbAreaReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	a := model.ZbbArea{
		Pid:         req.Pid,
		Code:        req.Code,
		Name:        req.Name,
		Fullname:    req.Fullname,
		Pinyin:      req.Pinyin,
		FirstLetter: req.FirstLetter,
		FullLetter:  req.FullLetter,
		Lat:         req.Lat,
		Lng:         req.Lng,
		Level:       req.Level,
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&a).Error == nil))
}

// GetAll GET /ZbbAreaBean/getAll 对齐 Java getAll：MyBatis-Plus 分页（无排序）。
func (c *ZbbAreaController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.ZbbArea{}).Count(&total)
	var rows []model.ZbbArea
	conf.Db.Model(&model.ZbbArea{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]zbbAreaDTO, 0, len(rows))
	for _, a := range rows {
		list = append(list, zbbAreaToDTO(a))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}
