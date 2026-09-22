package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
	"caicai-go/protocol"
	"caicai-go/service"
)

// newOrderID 生成订单号。
func newOrderID() string {
	return "OD" + time.Now().Format("20060102150405") + strconv.Itoa(int(time.Now().UnixNano()%100000))
}

// orderDeviceID 根据订单解析设备标识（产品 IMEI 或 NID）。
func orderDeviceID(orderID string) string {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil || order.Pid == "" {
		return ""
	}
	var prod model.TabProduct
	if conf.Db.Where("pid = ?", order.Pid).First(&prod).Error != nil {
		return ""
	}
	if prod.Imei != "" {
		return prod.Imei
	}
	return prod.Nid
}

// ============ 查询 ============

func (o *OrderController) GetByOrderId(c *gin.Context) {
	var order model.OrderTbl
	if err := conf.Db.Where("orderid = ?", c.Query("orderId")).First(&order).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderToDTO(order)))
}

func (o *OrderController) GetAllOrders(c *gin.Context) {
	var orders []model.OrderTbl
	conf.Db.Where("openid = ?", c.Query("openid")).Order("reserve_time desc").Find(&orders)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

func (o *OrderController) GetOrdersByPage(c *gin.Context) {
	current, size := parsePage(c)
	openid := c.Query("openid")
	var total int64
	conf.Db.Model(&model.OrderTbl{}).Where("openid = ?", openid).Count(&total)
	var orders []model.OrderTbl
	conf.Db.Where("openid = ?", openid).Order("reserve_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&orders)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (o *OrderController) GetOngoingOrders(c *gin.Context) {
	var orders []model.OrderTbl
	conf.Db.Where("openid = ? AND state IN ('使用中','已预约','预约中')", c.Query("openid")).Find(&orders)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

func (o *OrderController) GetDoneOrders(c *gin.Context) {
	var orders []model.OrderTbl
	conf.Db.Where("openid = ? AND state = '已完成'", c.Query("openid")).Find(&orders)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

func (o *OrderController) GetWaitPayOrders(c *gin.Context) {
	var orders []model.OrderTbl
	conf.Db.Where("openid = ? AND state = '待支付'", c.Query("openid")).Find(&orders)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

func (o *OrderController) GetOrdersByPageForCaiCai(c *gin.Context) {
	current, size := parsePage(c)
	openid := c.Query("openid")
	var total int64
	conf.Db.Model(&model.OrderTbl{}).Where("openid = ? AND consumption_type = '踩踩停车'", openid).Count(&total)
	var orders []model.OrderTbl
	conf.Db.Where("openid = ? AND consumption_type = '踩踩停车'", openid).Order("reserve_time desc").Offset(pageOffset(current, size)).Limit(size).Find(&orders)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (o *OrderController) GetNotFinishBySpacesId(c *gin.Context) {
	var orders []model.OrderTbl
	conf.Db.Where("spaces_id = ? AND state NOT IN ('已完成','已取消')", c.Query("spacesId")).Find(&orders)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

func (o *OrderController) GetWaitReturnGunOrderBySpacesId(c *gin.Context) {
	var order model.OrderTbl
	if err := conf.Db.Where("spaces_id = ? AND state = '待还枪'", c.Query("spacesId")).First(&order).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderToDTO(order)))
}

func (o *OrderController) GetOrderState(c *gin.Context) {
	var state string
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", c.Query("orderId")).Pluck("state", &state)
	c.JSON(http.StatusOK, ResultSuccess(state))
}

func (o *OrderController) GetplaceUsable(c *gin.Context) {
	var n int64
	conf.Db.Model(&model.OrderTbl{}).Where("lockid = ? AND state IN ('使用中','进行中','预约中','已预约')", c.Query("lockId")).Count(&n)
	if n > 0 {
		c.JSON(http.StatusOK, ResultSuccess("不可用"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("可使用"))
}

// ============ 下单/预约 ============

func (o *OrderController) Add(c *gin.Context) {
	order := model.OrderTbl{
		Orderid:  newOrderID(),
		Openid:   c.GetString("openid"),
		SpacesID: int32(queryInt(c, "spacesId", 0)),
		PlateNum: c.Query("plateNum"),
		State:    "预约中",
	}
	conf.Db.Create(&order)
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByLock(c *gin.Context) {
	order := model.OrderTbl{
		Orderid:  c.Query("orderid"),
		Openid:   c.Query("openid"),
		Lockid:   c.Query("lockid"),
		PlateNum: c.Query("plateNum"),
		State:    "预约中",
	}
	if order.Orderid == "" {
		order.Orderid = newOrderID()
	}
	conf.Db.Create(&order)
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByCharge(c *gin.Context) {
	order := model.OrderTbl{
		Orderid:         newOrderID(),
		Openid:          c.GetString("openid"),
		Pid:             c.Query("pid"),
		ConsumptionType: "驿享充电",
		State:           "待支付",
	}
	conf.Db.Create(&order)
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByTwiceCharge(c *gin.Context) {
	order := model.OrderTbl{
		Orderid:         newOrderID(),
		Openid:          c.GetString("openid"),
		Pid:             c.Query("pid"),
		ConsumptionType: "二轮车充电",
		TwiceChargeTime: c.Query("ChargeTime"),
		State:           "待支付",
	}
	conf.Db.Create(&order)
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByPrivateCharge(c *gin.Context) {
	order := model.OrderTbl{
		Orderid:         newOrderID(),
		Openid:          c.GetString("openid"),
		SpacesID:        int32(queryInt(c, "privatePlaceId", 0)),
		ConsumptionType: "邻享充电",
		State:           "待支付",
	}
	conf.Db.Create(&order)
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) Booking(c *gin.Context) {
	order := model.OrderTbl{
		Orderid:          newOrderID(),
		Openid:           c.GetString("openid"),
		SpacesID:         int32(queryInt(c, "spacesId", 0)),
		State:            "已预约",
		BookingStartTime: parseTimePtr(c.Query("startTime")),
		BookingEndTime:   parseTimePtr(c.Query("endTime")),
	}
	conf.Db.Create(&order)
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) BookingByYiXiang(c *gin.Context) {
	order := model.OrderTbl{
		Orderid:          newOrderID(),
		Openid:           c.GetString("openid"),
		SpacesID:         int32(queryInt(c, "spacesId", 0)),
		ConsumptionType:  "驿享充电",
		State:            "已预约",
		BookingStartTime: parseTimePtr(c.Query("startTime")),
		BookingEndTime:   parseTimePtr(c.Query("endTime")),
		TwiceChargeTime:  c.Query("ChargeTime"),
	}
	conf.Db.Create(&order)
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) NewBookingByYiXiang(c *gin.Context) {
	order := model.OrderTbl{
		Orderid:         newOrderID(),
		Openid:          c.GetString("openid"),
		SpacesID:        int32(queryInt(c, "spacesId", 0)),
		ConsumptionType: "驿享充电",
		State:           "预约中",
	}
	conf.Db.Create(&order)
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) CancelOrder(c *gin.Context) {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", c.Query("orderId")).Update("state", "已取消")
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (o *OrderController) Close(c *gin.Context) {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", c.Query("orderId")).Update("state", "已关闭")
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (o *OrderController) UpdateOrderState(c *gin.Context) {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", c.Query("orderId")).Update("state", "已支付")
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ 锁/充电控制（MQTT） ============

func (o *OrderController) StartLock(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(service.StartLock(c.Query("orderId"))))
}

func (o *OrderController) StopLock(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(service.StopLock(c.Query("orderId"))))
}

func (o *OrderController) Start(c *gin.Context) {
	dev := orderDeviceID(c.Query("orderId"))
	if dev == "" {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(service.SendCommand(dev, protocol.OpOpenCharge, protocol.DirectionDefault, nil)))
}

func (o *OrderController) Stop(c *gin.Context) {
	dev := orderDeviceID(c.Query("orderId"))
	if dev == "" {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(service.SendCommand(dev, protocol.OpCloseEle, protocol.DirectionDefault, nil)))
}

func (o *OrderController) OpenRelay(c *gin.Context) {
	dev := orderDeviceID(c.Query("orderId"))
	if dev == "" {
		c.JSON(http.StatusOK, ResultSuccess(""))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(strconv.FormatBool(service.SendCommand(dev, protocol.OpOpenGauge, protocol.DirectionDefault, nil))))
}

func (o *OrderController) CloseTwiceOrder(c *gin.Context) {
	dev := orderDeviceID(c.Query("orderId"))
	if dev != "" {
		service.SendCommand(dev, protocol.OpCloseGauge, protocol.DirectionDefault, nil)
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", c.Query("orderId")).Update("state", "已完成")
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) AsyncCloseTwiceGauge(c *gin.Context) {
	dev := orderDeviceID(c.Query("orderId"))
	if dev != "" {
		service.SendCommand(dev, protocol.OpCloseGauge, protocol.DirectionDefault, nil)
	}
	c.JSON(http.StatusOK, ResultSuccess(nil))
}

func (o *OrderController) CloseTwiceOrderWithoutGauge(c *gin.Context) {
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", c.Query("orderId")).Update("state", "已完成")
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ 微信支付/结算（stub） ============

func (o *OrderController) PayScore(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(gin.H{}))
}

func (o *OrderController) QueryPayScore(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(""))
}

func (o *OrderController) WechatCallback(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
}

func (o *OrderController) GetOrderPay(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(""))
}

func (o *OrderController) Finish(c *gin.Context) {
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", c.Query("orderId")).Update("state", "已完成")
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) FinishBalans(c *gin.Context) {
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", c.Query("orderId")).Update("state", "已完成")
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) Refunds(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(gin.H{}))
}

func (o *OrderController) CancelPayScore(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) UnfreezeOrder(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ 设备读数/其它 ============

func (o *OrderController) GetMeterValue(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess("0"))
}

func (o *OrderController) GetCurrentValue(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess("0"))
}

func (o *OrderController) GetVoltageValue(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess("0"))
}

func (o *OrderController) GetMeterValueByPrivate(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(gin.H{"power": 0, "current": 0, "voltage": 0}))
}

func (o *OrderController) GetOrderRechargeAmount(c *gin.Context) {
	totalFee, _ := electronicFeesByOrderid(c.Query("orderId"))
	c.JSON(http.StatusOK, ResultSuccess(totalFee))
}

func (o *OrderController) GetExpectDuration(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(0))
}

func (o *OrderController) SetOrderStateIsWaitReturnGun(c *gin.Context) {
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", c.Query("orderId")).Update("state", "待还枪")
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (o *OrderController) ManualDetectionIsReturnGun(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) ByOrderIdSendCloseEleAndMessage(c *gin.Context) {
	dev := orderDeviceID(c.Query("orderId"))
	if dev != "" {
		service.SendCommand(dev, protocol.OpCloseEle, protocol.DirectionDefault, nil)
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) AddImage(c *gin.Context) {
	// 简化：接收 file + phone，不做压缩/图片存储，仅返回成功
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// parseTimePtr 解析时间字符串为 time.Time（零值表示空）。
func parseTimePtr(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	s = strings.Replace(s, "T", " ", 1)
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}
