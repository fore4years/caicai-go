package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// hotelDTO 对应 Java domain.Hotel（hotel_tbl，驼峰）。
type hotelDTO struct {
	CreatTime     *LocalDateTime `json:"creatTime,omitempty"`
	Openid        string         `json:"openid,omitempty"`
	HotelName     string         `json:"hotelName,omitempty"`
	HotelAddress  string         `json:"hotelAddress,omitempty"`
	Phone         string         `json:"phone,omitempty"`
	Email         string         `json:"email,omitempty"`
	HotelTime     string         `json:"hotelTime,omitempty"`
	ID            int32          `json:"id"`
	ParkingNumber int32          `json:"parkingNumber,omitempty"`
	HotelImgID    int32          `json:"hotelImgId,omitempty"`
	Review        int32          `json:"review,omitempty"`
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

// List GET /system/hotel/list（对齐 Java HotelSerImpl.getHotelList：id 等值、openid 等值、
// hotelName 模糊、creat_time 倒序，返回 {list,total,pageNum,pageSize,pages}）。
func (a *AdminHotelController) List(c *gin.Context) {
	current, size := parsePage(c)
	id := c.Query("id")
	openid := c.Query("openid")
	hotelName := c.Query("hotelName")

	query := func() *gorm.DB {
		db := conf.Db.Model(&model.HotelTbl{})
		if id != "" {
			if v, err := strconv.Atoi(id); err == nil {
				db = db.Where("id = ?", v)
			}
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
	pages := (total + int64(size) - 1) / int64(size)
	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"list":     list,
		"total":    total,
		"pageNum":  current,
		"pageSize": size,
		"pages":    pages,
	}))
}

// UpdateReview PUT /system/hotel/updateReview?id=&review=
func (a *AdminHotelController) UpdateReview(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, "更新审核状态失败"))
		return
	}
	review, err := strconv.Atoi(c.Query("review"))
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, "更新审核状态失败"))
		return
	}
	res := conf.Db.Model(&model.HotelTbl{}).Where("id = ?", id).Update("review", review)
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
	if len(ids) == 0 {
		c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
		return
	}
	res := conf.Db.Where("id IN ?", ids).Delete(&model.HotelTbl{})
	if res.RowsAffected > 0 {
		c.JSON(http.StatusOK, ResultSuccess("批量删除成功"))
		return
	}
	c.JSON(http.StatusOK, ResultError(500, "批量删除失败"))
}
