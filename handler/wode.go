package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/objects"
)

// WodeController 对应 Java wxWodeCon（/wx/wode）。
type WodeController struct{}

// GetOrders 查看我的订单（对应 Java wxWodeSer.dingdanchaxun）。
func (w *WodeController) GetOrders(c *gin.Context) {
	openid := c.Query("openid")
	lueguo, _ := strconv.Atoi(c.DefaultQuery("lueguo", "0"))
	zhanshi, _ := strconv.Atoi(c.DefaultQuery("zhanshi", "10"))

	var orders []model.OrderTbl
	if err := conf.Db.Where("openid = ?", openid).Order("begin_time desc").Limit(zhanshi).Offset(lueguo).Find(&orders).Error; err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, []OrderDTO{})
		return
	}
	result := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		result = append(result, orderToDTO(o))
	}
	c.JSON(http.StatusOK, result)
}

// GetOrderXq 获取订单详情（对应 Java wxWodeSer.getOrderXq）。
func (w *WodeController) GetOrderXq(c *gin.Context) {
	orderid := c.Query("orderid")

	result := gin.H{}

	if order, err := objects.OrderTbl.WithContext(c.Request.Context()).Where(objects.OrderTbl.Orderid.Eq(orderid)).First(); err == nil {
		result["order"] = orderToDTO(*order)
	}

	// selectJion: order -> lock -> place
	var place model.PlaceTbl
	res := conf.Db.Raw(
		"SELECT p.* FROM order_tbl o JOIN lock_tbl l ON o.lockid = l.lockid JOIN place_tbl p ON p.placeid = l.placeid WHERE o.orderid = ?",
		orderid,
	).Scan(&place)
	if res.Error == nil && res.RowsAffected > 0 {
		result["place"] = placeToDTO(place)
	}

	c.JSON(http.StatusOK, result)
}

// Searchmsg 查询车牌号与手机号（对应 Java wxWodeSer.searchmsg）。
func (w *WodeController) Searchmsg(c *gin.Context) {
	openid := c.Query("openid")
	var msg struct {
		PlateNum string `json:"plate_num"`
		Phone    string `json:"phone"`
	}
	if err := conf.Db.Raw("SELECT plate_num, phone FROM user_tbl WHERE openid = ?", openid).Scan(&msg).Error; err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, msg)
}

// TimeDifference 比较当前时间与营业时间（对应 Java wxWodeSer.timeDifference）。
func (w *WodeController) TimeDifference(c *gin.Context) {
	placeid := c.Query("placeId")

	place, err := objects.PlaceTbl.WithContext(c.Request.Context()).Where(objects.PlaceTbl.Placeid.Eq(placeid)).First()
	if err != nil {
		c.JSON(http.StatusOK, Result{Success: false, Code: 500, Msg: "车位不存在"})
		return
	}

	if place.OpenTime == "全天" {
		c.JSON(http.StatusOK, ResultSuccess("ok"))
		return
	}

	parts := strings.Split(place.OpenTime, "-")
	if len(parts) < 2 {
		c.JSON(http.StatusOK, Result{Success: false, Code: 500, Msg: "此车位营业时间已过，请选择其他车位"})
		return
	}
	startTime, err1 := time.Parse("15:04", parts[0])
	endTime, err2 := time.Parse("15:04", parts[1])
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusOK, Result{Success: false, Code: 500, Msg: "此车位营业时间已过，请选择其他车位"})
		return
	}

	now := time.Now()
	startDateTime := time.Date(now.Year(), now.Month(), now.Day(), startTime.Hour(), startTime.Minute(), 0, 0, now.Location())
	endDateTime := time.Date(now.Year(), now.Month(), now.Day(), endTime.Hour(), endTime.Minute(), 0, 0, now.Location())

	if !(now.After(startDateTime) && now.Before(endDateTime)) {
		c.JSON(http.StatusOK, Result{Success: false, Code: 500, Msg: "此车位营业时间已过，请选择其他车位"})
		return
	}

	// 距离结束时间前 1 小时返回 fail
	oneHourBefore := endDateTime.Add(-time.Hour)
	if oneHourBefore.Hour() <= now.Hour() {
		c.JSON(http.StatusOK, ResultSuccess("fail"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("ok"))
}
