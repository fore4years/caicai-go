package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

func sysTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// sysTimeDiffMs 计算两个 "yyyy-MM-dd HH:mm:ss" 之间的毫秒差（end - start）。
func sysTimeDiffMs(start, end string) int {
	t1, err1 := time.ParseInLocation("2006-01-02 15:04:05", start, time.Local)
	t2, err2 := time.ParseInLocation("2006-01-02 15:04:05", end, time.Local)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(t2.Sub(t1).Milliseconds())
}

// ============ HotelController 自助侧（/hotel） ============

type HotelSelfController struct{}

// Add 对齐 Java HotelServiceImpl.add：更新已有酒店（要求已存在）。
func (c *HotelSelfController) Add(ctx *gin.Context) {
	var h model.HotelTbl
	if err := ctx.ShouldBindJSON(&h); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	var existing model.HotelTbl
	if err := conf.Db.Where("id = ?", h.ID).First(&existing).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "酒店封面未上传"))
		return
	}
	h.CreatTime = timePtr(time.Now())
	res := conf.Db.Model(&model.HotelTbl{}).Where("id = ?", h.ID).Updates(h)
	ctx.JSON(http.StatusOK, ResultSuccess(int(res.RowsAffected)))
}

func (c *HotelSelfController) GetByOpenid(ctx *gin.Context) {
	var rows []model.HotelTbl
	conf.Db.Where("openid = ?", ctx.GetString("openid")).Find(&rows)
	list := make([]hotelDTO, 0, len(rows))
	for _, h := range rows {
		list = append(list, hotelToDTO(h))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *HotelSelfController) GetById(ctx *gin.Context) {
	var h model.HotelTbl
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&h).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(hotelToDTO(h)))
}

// OpenLock 对齐 Java openLock：更新 hotel_product_tbl 使用状态（硬件指令省略）。
func (c *HotelSelfController) OpenLock(ctx *gin.Context) {
	lockID := ctx.Query("lockId")
	var hp model.HotelProductTbl
	if err := conf.Db.Where("productId = ?", lockID).First(&hp).Error; err == nil {
		now := sysTime()
		conf.Db.Model(&model.HotelProductTbl{}).Where("productId = ?", lockID).Updates(map[string]interface{}{
			"useState":   "1",
			"startTime":  now,
			"updateTime": now,
		})
	}
	ctx.JSON(http.StatusOK, ResultSuccess(1))
}

// CloseLock 对齐 Java closeLock：更新使用状态并累计使用时长。
func (c *HotelSelfController) CloseLock(ctx *gin.Context) {
	lockID := ctx.Query("lockId")
	var hp model.HotelProductTbl
	if err := conf.Db.Where("productId = ?", lockID).First(&hp).Error; err == nil {
		endTime := sysTime()
		oldUseTime := 0
		if hp.UseTime != "" {
			oldUseTime, _ = strconv.Atoi(hp.UseTime)
		}
		currentUseTimeMs := sysTimeDiffMs(hp.StartTime, endTime)
		currentUseTimeMinutes := (currentUseTimeMs + 59999) / 60000
		conf.Db.Model(&model.HotelProductTbl{}).Where("productId = ?", lockID).Updates(map[string]interface{}{
			"useState":   "0",
			"updateTime": endTime,
			"endTime":    endTime,
			"useTime":    strconv.Itoa(oldUseTime + currentUseTimeMinutes),
		})
	}
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// GetStaffInfo 对齐 Java getStaffInfo：按 hotelId + openId 查询单个员工。
func (c *HotelSelfController) GetStaffInfo(ctx *gin.Context) {
	var s model.HotelStaffTbl
	if err := conf.Db.Where("hotelId = ? AND openId = ?", ctx.Query("hotelId"), ctx.Query("openId")).First(&s).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(s))
}
