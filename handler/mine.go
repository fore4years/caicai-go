package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/logger"
	"caicai-go/repository"
)

// WodeController 对应 Java wxWodeCon（/wx/wode）。
type WodeController struct{}

// GetOrders 查看我的订单（对应 Java wxWodeSer.dingdanchaxun）。
func (w *WodeController) GetOrders(c *gin.Context) {
	openid := c.Query("openid")
	lueguo, _ := strconv.Atoi(c.DefaultQuery("lueguo", "0"))
	zhanshi, _ := strconv.Atoi(c.DefaultQuery("zhanshi", "10"))

	orders, err := repository.OrdersByOpenid(openid, lueguo, zhanshi)
	if err != nil {
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
// Java 的 map 始终包含 "order" 与 "place" 两个 key，查不到时为 null。
func (w *WodeController) GetOrderXq(c *gin.Context) {
	orderid := c.Query("orderid")

	result := gin.H{"order": nil, "place": nil}

	if order, err := repository.OrderByOrderid(c.Request.Context(), orderid); err == nil {
		result["order"] = orderToDTO(*order)
	}

	// selectJion: order -> lock -> place
	if place, ok := repository.PlaceByOrderidJoin(orderid); ok {
		result["place"] = placeToDTO(*place)
	}

	c.JSON(http.StatusOK, result)
}

// Searchmsg 查询车牌号与手机号（对应 Java wxWodeSer.searchmsg）。
func (w *WodeController) Searchmsg(c *gin.Context) {
	openid := c.Query("openid")
	plateNum, phone, err := repository.UserPlateAndPhone(openid)
	if err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"plate_num": plateNum, "phone": phone})
}

// TimeDifference 比较当前时间与营业时间（对应 Java wxWodeSer.timeDifference）。
func (w *WodeController) TimeDifference(c *gin.Context) {
	placeid := c.Query("placeId")

	place, err := repository.PlaceByPlaceid(c.Request.Context(), placeid)
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
		c.JSON(http.StatusOK, Result{Success: false, Code: 400, Msg: "此车位营业时间已过，请选择其他车位"})
		return
	}
	startTime, err1 := time.Parse("15:04", parts[0])
	endTime, err2 := time.Parse("15:04", parts[1])
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusOK, Result{Success: false, Code: 400, Msg: "此车位营业时间已过，请选择其他车位"})
		return
	}

	now := time.Now()
	startDateTime := time.Date(now.Year(), now.Month(), now.Day(), startTime.Hour(), startTime.Minute(), 0, 0, now.Location())
	endDateTime := time.Date(now.Year(), now.Month(), now.Day(), endTime.Hour(), endTime.Minute(), 0, 0, now.Location())

	if !(now.After(startDateTime) && now.Before(endDateTime)) {
		c.JSON(http.StatusOK, Result{Success: false, Code: 400, Msg: "此车位营业时间已过，请选择其他车位"})
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

// OverOrder POST /wx/wode/overOrder 对齐 Java wxWodeSerImpl.overOrder：
// 结束停车订单，按停车费率（30 分钟免费、按 30 分钟档计费 + 超时计费）计算金额，置为待支付。
func (w *WodeController) OverOrder(c *gin.Context) {
	var req struct {
		Orderid string `json:"orderid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, "ok")
		return
	}
	order, err := repository.FindOrderByOrderid(req.Orderid)
	if err != nil {
		c.JSON(http.StatusOK, "ok")
		return
	}
	place := repository.PlaceByLockid(order.Lockid)
	if place == nil {
		c.JSON(http.StatusOK, "ok")
		return
	}

	now := time.Now()
	var totalPrice decimal.Decimal
	if !isZeroTime(order.BeginTime) {
		useTime := now.Sub(timeOrZero(order.BeginTime))
		minutes := int(useTime.Minutes())
		if minutes <= 30 {
			totalPrice = decimal.Zero
		} else {
			extraMinutes := minutes - 30
			intervals := extraMinutes / 30
			if extraMinutes%30 > 0 {
				intervals++
			}
			rate, _ := decimal.NewFromString(place.Rate)
			totalPrice = rate.Mul(decimal.NewFromInt(int64(intervals)))
		}

		openTime := place.OpenTime
		if openTime != "" && openTime != "全天" {
			parts := strings.Split(openTime, "-")
			if len(parts) >= 2 {
				closeTimeStr := order.BeginTime.Format("2006-01-02") + " " + parts[1] + ":00"
				if closeT, err := time.ParseInLocation("2006-01-02 15:04:05", closeTimeStr, time.Local); err == nil {
					okTime := closeT.Sub(timeOrZero(order.BeginTime))
					if useTime > okTime {
						availableMinutes := int(okTime.Minutes())
						useMinutes := minutes
						rate, _ := decimal.NewFromString(place.Rate)
						if availableMinutes >= 30 {
							availableMinutes -= 30
							useMinutes -= 30
							totalPrice = decimal.NewFromFloat(float64(useMinutes-availableMinutes) * 0.5).Add(rate)
						} else {
							useMinutes -= availableMinutes
							totalPrice = decimal.NewFromFloat(float64(useMinutes) * 0.5).Add(rate)
						}
					}
				}
				if totalPrice.GreaterThan(decimal.NewFromInt(200)) {
					totalPrice = decimal.NewFromInt(200)
				}
			}
		}
	}

	// Java orderService.stopLock(orderid) 下发关锁指令，省略。
	repository.UpdatePlaceState(place.Placeid, "可使用")

	chargingTime := ""
	if !isZeroTime(order.BeginTime) {
		d := now.Sub(timeOrZero(order.BeginTime))
		chargingTime = time.Time{}.Add(d).Format("15:04:05")
	}
	repository.UpdateOrderFields(req.Orderid, map[string]interface{}{
		"over_time":         now,
		"close_time":        now,
		"total_price":       totalPrice,
		"basic_consumption": totalPrice,
		"state":             repository.OrderStateWaitPay,
		"charging_time":     chargingTime,
		"stop_mode":         "扫码自停",
	})
	// 微信订阅消息推送省略（Java subscribeMessageService.ParkingMessage）。
	c.JSON(http.StatusOK, "ok")
}

// ChargeOrderPay POST /wx/wode/chargeOrderPay 对齐 Java wxWodeSerImpl.chargeOrderPay：
// charge_order_tbl 置为已完成（Java CaiCaiPayUtil.isPaySuccess() 恒为 true，外呼省略）。
func (w *WodeController) ChargeOrderPay(c *gin.Context) {
	var req struct {
		Orderid string `json:"orderid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, "fail")
		return
	}
	o, err := repository.FindChargeOrderByOrderid(req.Orderid)
	if err != nil {
		c.JSON(http.StatusOK, "fail")
		return
	}
	// CaiCaiPayUtil.isPaySuccess() 恒为 true；Java updataByOrderid 会回写其它同值字段（等价于只改 state）。
	repository.UpdateChargeOrderState(o.Orderid, "已完成")
	c.JSON(http.StatusOK, "ok")
}
