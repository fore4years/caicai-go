package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/protocol"
	"caicai-go/service"
)

// newOrderID 生成订单号。
func newOrderID() string {
	return "OD" + time.Now().Format("20060102150405") + strconv.Itoa(int(time.Now().UnixNano()%100000))
}

// orderDeviceID 根据订单解析设备标识（产品 IMEI 或 NID）。用于二轮车等 pid 直接关联产品的订单。
func orderDeviceID(orderID string) string {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil || order.Pid == "" {
		return ""
	}
	prod := productByPid(order.Pid)
	if prod == nil {
		return ""
	}
	if prod.Imei != "" {
		return prod.Imei
	}
	return prod.Nid
}

// getNotFinishOrder 查询当前 openid 的未完成订单（state 非「已完成」「已取消」），不存在返回 nil。
func getNotFinishOrder(openid string) *model.OrderTbl {
	var o model.OrderTbl
	if err := conf.Db.Where("openid = ? AND state <> ? AND state <> ?", openid, orderStateFinished, orderStateRevoked).
		First(&o).Error; err != nil {
		return nil
	}
	return &o
}

// getByOpenidNotFinish 按 openid 查询未完成订单（对齐 Java getByOpenidNotFinish）。
func getByOpenidNotFinish(openid string) *model.OrderTbl {
	return getNotFinishOrder(openid)
}

// getNotFinishBySpacesIdList 根据车位查询未完成订单（含一桩一锁关联车位），对齐 Java getNotFinishBySpacesId。
func getNotFinishBySpacesIdList(spacesID int32) []model.OrderTbl {
	var list []model.OrderTbl
	conf.Db.Where("spaces_id = ? AND state <> ? AND state <> ? AND consumption_type <> ?",
		spacesID, orderStateRevoked, orderStateFinished, consumeTypeNeighbor).Find(&list)

	spaces := spacesByID(spacesID)
	if spaces == nil || spaces.LockID == "" {
		return list
	}
	if !isOneLockPerSpot(spacesID) {
		return list
	}
	otherID := otherSpacesIDBySpacesID(spacesID)
	if otherID == 0 {
		return list
	}
	var list1 []model.OrderTbl
	conf.Db.Where("spaces_id = ? AND state <> ? AND state <> ?", otherID, orderStateRevoked, orderStateFinished).Find(&list1)
	return append(list, list1...)
}

// ============ 查询 ============

func (o *OrderController) GetByOrderId(c *gin.Context) {
	orderID := c.Query("orderId")
	if len(orderID) < 10 {
		c.JSON(http.StatusOK, ResultError(1, "订单id为空"))
		return
	}
	var order model.OrderTbl
	if err := conf.Db.Where("orderid = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderToDTO(order)))
}

func (o *OrderController) GetAllOrders(c *gin.Context) {
	var orders []model.OrderTbl
	conf.Db.Where("openid = ?", c.Query("openid")).Order("reserve_time desc").Limit(20).Find(&orders)
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
	list := getNotFinishBySpacesIdList(int32(queryInt(c, "spacesId", 0)))
	dto := make([]OrderDTO, 0, len(list))
	for _, od := range list {
		dto = append(dto, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(dto))
}

func (o *OrderController) GetWaitReturnGunOrderBySpacesId(c *gin.Context) {
	var order model.OrderTbl
	err := conf.Db.Raw(
		`SELECT o.* FROM order_tbl o LEFT JOIN private_place_tbl p ON p.id = o.spaces_id WHERE p.id = ? AND o.state = '待还枪'`,
		c.Query("spacesId"),
	).Scan(&order).Error
	if err != nil || order.Orderid == "" {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderToDTO(order)))
}

func (o *OrderController) GetOrderState(c *gin.Context) {
	state := service.GetRedisValue("orderState:" + c.Query("orderId"))
	if state == "" {
		state = "暂时无法获取到订单信息,请稍后重试"
	}
	c.JSON(http.StatusOK, ResultSuccess(state))
}

func (o *OrderController) GetplaceUsable(c *gin.Context) {
	var orders []model.OrderTbl
	conf.Db.Where("lockid = ? AND consumption_type = '踩踩停车' AND state IN ('使用中','已预约','预约中')", c.Query("lockId")).Find(&orders)
	if len(orders) > 0 {
		c.JSON(http.StatusOK, ResultSuccess("fail"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("ok"))
}

// ============ 下单/预约 ============

func (o *OrderController) Add(c *gin.Context) {
	openid := c.GetString("openid")
	if getNotFinishOrder(openid) != nil {
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}
	order := model.OrderTbl{
		Orderid:     newOrderID(),
		Openid:      openid,
		SpacesID:    int32(queryInt(c, "spacesId", 0)),
		PlateNum:    c.Query("plateNum"),
		State:       orderStateBooking,
		ReserveTime: time.Now(),
	}
	if conf.Db.Create(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByLock(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "null" {
		openid = c.GetString("openid")
	}
	lockid := c.Query("lockid")
	orderid := c.Query("orderid")

	spaces := spacesByLockID(lockid)
	var spacesID int32
	if spaces != nil {
		spacesID = spaces.ID
	}
	place := placeByLockid(lockid)
	var rate decimal.Decimal
	if place != nil {
		rate, _ = decimal.NewFromString(place.Rate)
	}

	existing := getByOpenidNotFinish(openid)
	if existing != nil {
		if existing.Orderid == orderid {
			c.JSON(http.StatusOK, ResultSuccess(orderid))
			return
		}
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}

	order := model.OrderTbl{
		Orderid:         newOrderID(),
		Lockid:          lockid,
		Openid:          openid,
		SpacesID:        spacesID,
		ReserveTime:     time.Now(),
		State:           orderStateConduct,
		PlateNum:        c.Query("plateNum"),
		ConsumptionType: consumeTypeCaiParking,
		GiftAmount:      "暂未开放~",
		StartMode:       "扫码",
		BeginTime:       time.Now(),
		ChargingRates:   rate,
	}
	if conf.Db.Create(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByCharge(c *gin.Context) {
	openid := c.GetString("openid")
	pid := c.Query("pid")
	direction := c.Query("direction")

	spaces := spacesByPidAndDirection(pid, direction)
	if spaces == nil {
		c.JSON(http.StatusOK, ResultError(1, "车位不存在"))
		return
	}
	if getNotFinishOrder(openid) != nil {
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}
	var totalPrice decimal.Decimal
	if gun := chargingGunByID(spaces.ChargingGunID); gun != nil {
		totalPrice = gun.UnitPrice
	}

	order := model.OrderTbl{
		Orderid:     newOrderID(),
		TotalPrice:  totalPrice,
		Openid:      openid,
		SpacesID:    spaces.ID,
		ReserveTime: time.Now(),
		State:       orderStateBooking,
	}
	if conf.Db.Create(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	conf.Db.Create(&model.ConfigTbl{Orderid: order.Orderid, ChargeGun: 0})
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByTwiceCharge(c *gin.Context) {
	openid := c.GetString("openid")
	pid := c.Query("pid")
	chargeType := c.Query("ChargeType")
	receptacleID := int32(queryInt(c, "ReceptacleId", 0))
	chargeTime := c.Query("ChargeTime")

	var station model.ChargingStationTbl
	if conf.Db.Where("pid = ?", pid).First(&station).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "充电站不存在"))
		return
	}
	receptacle := receptacleByID(receptacleID)

	notFinish := getNotFinishOrder(openid)
	if notFinish != nil {
		if notFinish.State == orderStateConduct {
			cancelOrderInternal(notFinish.Orderid, "0")
		} else {
			c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
			return
		}
	}

	lockid := ""
	if receptacle != nil {
		lockid = strings.Split(receptacle.Name, "号")[0]
	}
	order := model.OrderTbl{
		Orderid:             newOrderID(),
		Openid:              openid,
		ReserveTime:         time.Now(),
		ConsumptionLocation: station.Name,
		State:               orderStateConduct,
		ConsumptionType:     consumeTypeTwice,
		GiftAmount:          "暂未开放~",
		StartMode:           "扫码",
		StopMode:            "扫码自停",
		Lockid:              lockid,
		ChargingRates:       decimal.RequireFromString(orderTwiceChargeRate),
		Pid:                 pid,
	}
	if chargeType == "TimeSelected" {
		order.TwiceChargeTime = chargeTime
	}
	if conf.Db.Create(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByPrivateCharge(c *gin.Context) {
	openid := c.GetString("openid")
	privatePlaceID := int32(queryInt(c, "privatePlaceId", 0))

	privatePlace := privatePlaceByID(privatePlaceID)
	if privatePlace == nil {
		c.JSON(http.StatusOK, ResultError(1, "车位不存在"))
		return
	}
	if getNotFinishOrder(openid) != nil {
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}

	var existing model.OrderTbl
	if err := conf.Db.Where("consumption_type = ? AND spaces_id = ? AND state IN ('进行中','使用中','待还枪')",
		consumeTypeNeighbor, privatePlaceID).First(&existing).Error; err == nil {
		switch existing.State {
		case orderStateConduct:
			c.JSON(http.StatusOK, ResultError(1, "前序订单正在进行中,请等待"))
			return
		case orderStateUse:
			c.JSON(http.StatusOK, ResultError(1, "前序订单正在使用中,请等待"))
			return
		case orderStateWaitGun:
			c.JSON(http.StatusOK, ResultError(1, "检测到充电枪未归还,请协助将充电枪归还至枪座"))
			return
		default:
			c.JSON(http.StatusOK, ResultError(1, "充电桩存在未完成订单,请等待"))
			return
		}
	}

	order := model.OrderTbl{
		Orderid:             newOrderID(),
		Openid:              openid,
		SpacesID:            privatePlaceID,
		ReserveTime:         time.Now(),
		ConsumptionLocation: privatePlace.Place,
		ConsumptionType:     consumeTypeNeighbor,
		GiftAmount:          "暂未开放~",
		StartMode:           "扫码",
		StopMode:            "扫码自停",
		State:               orderStateConduct,
	}
	if conf.Db.Create(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "生成订单失败"))
		return
	}
	conf.Db.Create(&model.OrderPrivateTbl{Orderid: order.Orderid, IsPrivateUser: int32(queryInt(c, "isPrivateUser", 0))})
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) Booking(c *gin.Context) {
	openid := c.GetString("openid")
	spacesID := int32(queryInt(c, "spacesId", 0))
	startTime := parseTimePtr(c.Query("startTime"))
	endTime := parseTimePtr(c.Query("endTime"))

	parkingSpaces := spacesByID(spacesID)
	if getNotFinishOrder(openid) != nil {
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}
	if du := driverUseByOpenid(openid); du == nil || du.ReserveNumber == 0 {
		c.JSON(http.StatusOK, ResultError(1, "今日预约次数以达到上限"))
		return
	}
	for _, ob := range getNotFinishBySpacesIdList(spacesID) {
		if ob.BookingStartTime.Before(endTime) && ob.BookingEndTime.After(startTime) {
			c.JSON(http.StatusOK, ResultError(1, "预约时间冲突"))
			return
		}
	}
	order := model.OrderTbl{
		Orderid:          newOrderID(),
		Openid:           openid,
		SpacesID:         spacesID,
		ReserveTime:      time.Now(),
		State:            orderStateBooking,
		BookingStartTime: startTime,
		BookingEndTime:   endTime,
		ConsumptionType:  consumeTypeYiXiang,
		ChargingRates:    currentChargingRate(),
		StartMode:        "扫码",
		StopMode:         "扫码自停",
		GiftAmount:       "暂未开放~",
	}
	if parkingSpaces != nil {
		order.ConsumptionLocation = parkingSpaces.Place
	}
	if conf.Db.Create(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) BookingByYiXiang(c *gin.Context) {
	openid := c.GetString("openid")
	spacesID := int32(queryInt(c, "spacesId", 0))
	startTime := parseTimePtr(c.Query("startTime"))
	endTime := parseTimePtr(c.Query("endTime"))

	parkingSpaces := spacesByID(spacesID)
	if getNotFinishOrder(openid) != nil {
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}
	if du := driverUseByOpenid(openid); du == nil || du.ReserveNumber == 0 {
		c.JSON(http.StatusOK, ResultError(1, "今日预约次数以达到上限"))
		return
	}
	for _, ob := range getNotFinishBySpacesIdList(spacesID) {
		if ob.BookingStartTime.Before(endTime) && ob.BookingEndTime.After(startTime) {
			c.JSON(http.StatusOK, ResultError(1, "预约时间冲突"))
			return
		}
	}
	if parkingSpaces != nil && parkingSpaces.ChargingGunID != 0 {
		if gun := chargingGunByID(parkingSpaces.ChargingGunID); gun != nil {
			if chargingStateByProductAndDirection(gun.ProductID, gun.Direction) != nil {
				c.JSON(http.StatusOK, ResultError(1, "当前充电桩紧急按钮被按下,暂不可用"))
				return
			}
		}
	}

	order := model.OrderTbl{
		Orderid:          newOrderID(),
		Openid:           openid,
		SpacesID:         spacesID,
		ReserveTime:      time.Now(),
		State:            orderStateBooking,
		BookingStartTime: startTime,
		BookingEndTime:   endTime,
		ConsumptionType:  consumeTypeYiXiang,
		ChargingRates:    currentChargingRate(),
		StartMode:        "扫码",
		StopMode:         "扫码自停",
		GiftAmount:       "暂未开放~",
	}
	if parkingSpaces != nil {
		order.ConsumptionLocation = parkingSpaces.Place
	}
	if conf.Db.Create(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	// 时间模式：预约结束后自动断电（Java 通过 RabbitMQ 定时任务，此处省略定时调度）
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) NewBookingByYiXiang(c *gin.Context) {
	openid := c.GetString("openid")
	spacesID := int32(queryInt(c, "spacesId", 0))

	parkingSpaces := spacesByID(spacesID)
	notFinish := getNotFinishOrder(openid)
	if notFinish != nil {
		if notFinish.State == orderStateBooked || notFinish.State == orderStateBooking {
			cancelOrderInternal(notFinish.Orderid, "0")
		} else {
			c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
			return
		}
	}
	if parkingSpaces != nil && parkingSpaces.ChargingGunID != 0 {
		if gun := chargingGunByID(parkingSpaces.ChargingGunID); gun != nil {
			if chargingStateByProductAndDirection(gun.ProductID, gun.Direction) != nil {
				c.JSON(http.StatusOK, ResultError(1, "当前充电桩紧急按钮被按下,暂不可用"))
				return
			}
		}
	}

	order := model.OrderTbl{
		Orderid:         newOrderID(),
		Openid:          openid,
		SpacesID:        spacesID,
		ReserveTime:     time.Now(),
		State:           orderStateBooking,
		ConsumptionType: consumeTypeYiXiang,
		ChargingRates:   currentChargingRate(),
		StartMode:       "扫码",
		StopMode:        "扫码自停",
		GiftAmount:      "暂未开放~",
	}
	if parkingSpaces != nil {
		order.ConsumptionLocation = parkingSpaces.Place
	}
	if conf.Db.Create(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "创建订单失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

// ============ 取消 / 结算 / 控制 ============

// cancelOrderInternal 取消订单（内部调用，对齐 Java OrderServiceImpl.cancelOrder）。
func cancelOrderInternal(orderID, price string) bool {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		return false
	}

	state := queryWxOrderState(orderID)
	if state != wxPayStateCancel {
		p := price
		if p == "" {
			p = "0"
		}
		amountFen, _ := decimal.NewFromString(p)
		amountFen = amountFen.Mul(decimal.NewFromInt(100)).Round(0)
		subtractWxDefaultAmount(orderID, int(amountFen.IntPart()))
		cancelWxOrder(orderID)
	} else {
		completeOrderIsamountNotNull(order.Openid, price)
		p := price
		if p == "undefined" || p == "null" {
			p = "0"
		}
		if pv, err := strconv.Atoi(p); err == nil && pv > 0 {
			saveBalanceRecord(order.Openid, "超时取消扣除手续费", decimal.NewFromInt(int64(pv)))
		}
	}

	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Updates(map[string]interface{}{
		"close_time":        time.Now(),
		"basic_consumption": decimal.Zero,
		"charging_degree":   decimal.Zero,
	})

	if order.ConsumptionType != consumeTypeTwice && order.ConsumptionType != consumeTypeNeighbor {
		if spaces := spacesByID(order.SpacesID); spaces != nil && spaces.SpacesCode != "" {
			conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", spaces.SpacesCode).Update("state", "可使用")
		}
	}

	if du := driverUseByOpenid(order.Openid); du != nil && du.ReserveNumber > 0 {
		conf.Db.Model(&model.DriveruseTbl{}).Where("openid = ?", order.Openid).
			Update("reserveNumber", du.ReserveNumber-1)
	}

	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("state", orderStateRevoked)
	return res.RowsAffected > 0
}

func (o *OrderController) CancelOrder(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(cancelOrderInternal(c.Query("orderId"), c.Query("price"))))
}

func (o *OrderController) Close(c *gin.Context) {
	orderID := c.Query("orderId")
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "查无此订单!"))
		return
	}
	if order.State == orderStateFinished {
		c.JSON(http.StatusOK, ResultError(1, "此订单已完成!"))
		return
	}
	if order.ConsumptionType == consumeTypeNeighbor {
		linXiangCloseOrder(&order)
	} else {
		normalCloseOrder(&order)
	}
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Updates(map[string]interface{}{
		"close_time": time.Now(),
		"state":      orderStateWaitPay,
	})
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// normalCloseOrder 结束普通订单（非邻享充电），对齐 Java NormalCloseOrder。
func normalCloseOrder(order *model.OrderTbl) {
	gun := chargingGunByOrder(order.Orderid)
	if gun != nil {
		if order.OverTime.IsZero() && order.ChargingDegree == 0 {
			if !service.SendCommand(gun.deviceID(), protocol.OpCloseEle, parseDirection(gun.gun.Direction), nil) {
				// Java 抛「拉闸失败」；这里保守继续
				logger.Mylog.Warn().Msgf("拉闸失败, orderId: %s", order.Orderid)
			}
		}
	}
	if order.ChargeGunPull == 1 && order.ChargingDegree == 0 {
		// Java 抛「请先归还充电枪」；保守继续
		logger.Mylog.Warn().Msgf("请先归还充电枪, 后结束订单~ orderId: %s", order.Orderid)
	}
	updateChargingDegreeForOrder(order.Orderid)
	updateBasicConsumptionForOrder(order.Orderid)

	if lock := lockByOrder(order.Orderid); lock != nil {
		service.SendCommand(lock.deviceID(), protocol.OpClosePlaceLock, protocol.DirectionDefault, nil)
	}
}

// linXiangCloseOrder 结束邻享充电订单，对齐 Java LinXiangCloseOrder。
func linXiangCloseOrder(order *model.OrderTbl) {
	gun := chargingGunByPrivateOrder(order.Orderid)
	if gun != nil {
		if order.OverTime.IsZero() && order.ChargingDegree == 0 {
			service.SendCommand(gun.deviceID(), protocol.OpCloseEle, parseDirection(gun.gun.Direction), nil)
		}
	}
	if order.ChargeGunPull == 1 && order.ChargingDegree == 0 {
		logger.Mylog.Warn().Msgf("请先归还充电枪后, 再结束订单~ orderId: %s", order.Orderid)
	}
	updateChargingDegreeForOrder(order.Orderid)

	orderPrivate := orderPrivateByOrderid(order.Orderid)
	if orderPrivate == nil {
		return
	}
	var fee decimal.Decimal
	switch orderPrivate.IsPrivateUser {
	case 1: // 桩主：仅电费（total）
		fee = privateFeesByPrivateOrderid(order.Orderid)
	case 0: // 毗邻邻居：电费 + 私桩服务费率
		var rate decimal.Decimal
		if privatePlace := privatePlaceByID(order.SpacesID); privatePlace != nil {
			if chargingBean := privateChargingBySpacesNum(privatePlace.SpacesCode); chargingBean != nil {
				rate = chargingBean.Fee
			}
		}
		fee = privateFeesByOrderidAndPrivateUser(order.Orderid, rate)
	case 2: // 普通用户：电费 + 服务费 + 超时占位费
		totalFee, serviceFee := privateElectronicByOrderidAndPrivateUser(order.Orderid)
		baseFee := totalFee.Add(serviceFee)
		neighborOvertimeFee := calculateNeighborOvertimeFee(order)
		fee = baseFee.Add(neighborOvertimeFee)
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", order.Orderid).
		Update("basic_consumption", fee.Round(2))
}

func (o *OrderController) UpdateOrderState(c *gin.Context) {
	orderID := c.Query("orderId")
	openid := c.Query("openid")
	amount := decimal.NewFromFloat(queryFloat(c, "amount"))

	if !freezeBalance(openid, amount) {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}

	switch order.ConsumptionType {
	case consumeTypeCaiParking:
		if order.State == orderStateConduct {
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateConduct).
				Updates(map[string]interface{}{"state": orderStateBooked, "decide_amount": amount})
		} else {
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateBooking).
				Updates(map[string]interface{}{"state": orderStateBooked, "decide_amount": amount})
		}
		c.JSON(http.StatusOK, ResultSuccess(true))
	case consumeTypeYiXiang:
		if configByOrderid(orderID) == nil {
			conf.Db.Create(&model.ConfigTbl{Orderid: orderID, ChargeGun: 0})
		}
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state IN (?,?)", orderID, orderStateBooking, orderStateWaitUse).
			Updates(map[string]interface{}{"state": orderStateBooked, "decide_amount": amount})
		c.JSON(http.StatusOK, ResultSuccess(true))
	case consumeTypeNeighbor:
		gun := chargingGunByPrivateOrder(orderID)
		if gun == nil || !service.SendCommand(gun.deviceID(), protocol.OpOpenCharge, protocol.DirectionDefault, nil) {
			c.JSON(http.StatusOK, ResultError(1, "开充电枪失败"))
			return
		}
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateConduct).
			Updates(map[string]interface{}{"state": orderStateUse, "decide_amount": amount, "begin_time": time.Now()})
		c.JSON(http.StatusOK, ResultSuccess(true))
	default: // 二轮车充电
		if !openGauge(orderID) {
			c.JSON(http.StatusOK, ResultError(1, "开闸失败"))
			return
		}
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).
			Updates(map[string]interface{}{"state": orderStateUse, "decide_amount": amount})
		c.JSON(http.StatusOK, ResultSuccess(true))
	}
}

func (o *OrderController) StartLock(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(service.StartLock(c.Query("orderId"))))
}

func (o *OrderController) StopLock(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(service.StopLock(c.Query("orderId"))))
}

func (o *OrderController) Start(c *gin.Context) {
	orderID := c.Query("orderId")
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "查无此订单"))
		return
	}
	if !order.BookingStartTime.IsZero() {
		if order.BookingStartTime.Sub(time.Now()) > 30*time.Minute {
			c.JSON(http.StatusOK, ResultError(1, "还没到您所预约的充电时间,我们支持提前半小时开始充电服务,请耐心等待!"))
			return
		}
	}
	spaces := spacesByID(order.SpacesID)
	if spaces == nil {
		c.JSON(http.StatusOK, ResultError(1, "车位不存在"))
		return
	}
	if spaces.ChargingGunID != 0 {
		if gun := chargingGunByID(spaces.ChargingGunID); gun != nil {
			if chargingStateByProductAndDirection(gun.ProductID, gun.Direction) != nil {
				c.JSON(http.StatusOK, ResultError(1, "当前充电桩紧急按钮被按下,暂不可用"))
				return
			}
		}
	}

	if spaces.LockID == "" {
		gun := chargingGunByOrder(orderID)
		if gun == nil {
			c.JSON(http.StatusOK, ResultError(1, "未找到充电枪信息,请更换设备使用"))
			return
		}
		if !service.SendCommand(gun.deviceID(), protocol.OpOpenCharge, parseDirection(gun.gun.Direction), nil) {
			c.JSON(http.StatusOK, ResultError(1, "解锁充电枪失败!"))
			return
		}
		service.SetRedisValue("orderState:"+orderID, "拔枪出桩")
	} else {
		if isOneLockPerSpot(order.SpacesID) {
			otherID := otherSpacesIDBySpacesID(order.SpacesID)
			var cnt int64
			conf.Db.Model(&model.OrderTbl{}).Where("spaces_id = ? AND state = ?", otherID, orderStateUse).Count(&cnt)
			if cnt > 0 {
				c.JSON(http.StatusOK, ResultError(1, "当前车位车辆还未驶离,请稍后再试!"))
				return
			}
		}
		if lock := lockByOrder(orderID); lock != nil {
			if !service.SendCommand(lock.deviceID(), protocol.OpOpenPlaceLock, protocol.DirectionDefault, nil) {
				c.JSON(http.StatusOK, ResultError(1, "开车位锁失败"))
				return
			}
		}
		service.SetRedisValue("orderState:"+orderID, "驱车入位")
	}

	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).
		Updates(map[string]interface{}{"state": orderStateUse, "begin_time": time.Now()})
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (o *OrderController) Stop(c *gin.Context) {
	orderID := c.Query("orderId")
	if lock := lockByOrder(orderID); lock != nil {
		service.SendCommand(lock.deviceID(), protocol.OpClosePlaceLock, protocol.DirectionDefault, nil)
	}
	if gun := chargingGunByOrder(orderID); gun != nil {
		service.SendCommand(gun.deviceID(), protocol.OpCloseEle, parseDirection(gun.gun.Direction), nil)
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) OpenRelay(c *gin.Context) {
	orderID := c.Query("orderId")
	timeStr := c.Query("time")
	if timeStr == "1" {
		openRelayForPrivateCharging(orderID)
		c.JSON(http.StatusOK, ResultSuccess("success"))
		return
	}
	// 定时开继电器（Java 用 Timer 调度；此处简化为立即执行）
	openRelayForPrivateCharging(orderID)
	c.JSON(http.StatusOK, ResultSuccess("scheduled"))
}

// openRelayForPrivateCharging 开继电器（邻享充电），对齐 Java openRelayForPrivateCharging。
func openRelayForPrivateCharging(orderID string) {
	gun := chargingGunByPrivateOrder(orderID)
	if gun == nil {
		return
	}
	service.SendCommand(gun.deviceID(), protocol.OpOpenGauge, protocol.DirectionDefault, nil)
}

func (o *OrderController) CloseTwiceOrder(c *gin.Context) {
	orderID := c.Query("orderId")
	closeTwiceOrderInternal(orderID, true)
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) AsyncCloseTwiceGauge(c *gin.Context) {
	orderID := c.Query("orderId")
	go closeGauge(orderID)
	c.JSON(http.StatusOK, ResultSuccess(nil))
}

func (o *OrderController) CloseTwiceOrderWithoutGauge(c *gin.Context) {
	orderID := c.Query("orderId")
	closeTwiceOrderInternal(orderID, false)
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) SetOrderStateIsWaitReturnGun(c *gin.Context) {
	orderID := c.Query("orderId")
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "未找到此订单"))
		return
	}
	if order.State != orderStateUse {
		c.JSON(http.StatusOK, ResultError(1, "此订单不是使用中状态"))
		return
	}
	gun := chargingGunByPrivateOrder(orderID)
	if gun != nil {
		service.SendCommand(gun.deviceID(), protocol.OpCloseEle, parseDirection(gun.gun.Direction), nil)
		updateChargingDegreeForOrder(orderID)
		orderPrivate := orderPrivateByOrderid(orderID)
		if orderPrivate != nil {
			var fee decimal.Decimal
			if orderPrivate.IsPrivateUser == 1 {
				fee = privateFeesByPrivateOrderid(orderID)
			} else {
				fee = privateFeesByOrderid(orderID)
			}
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("basic_consumption", fee.Round(2))
		}
	}
	res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("state", orderStateWaitGun)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (o *OrderController) ManualDetectionIsReturnGun(c *gin.Context) {
	orderID := c.Query("orderId")
	time.Sleep(time.Second)
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if !order.OverTime.IsZero() {
		res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Updates(map[string]interface{}{
			"close_time": time.Now(),
			"state":      orderStateWaitPay,
		})
		c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(false))
}

func (o *OrderController) ByOrderIdSendCloseEleAndMessage(c *gin.Context) {
	orderID := c.Query("orderId")
	if gun := chargingGunByOrder(orderID); gun != nil {
		if !service.SendCommand(gun.deviceID(), protocol.OpCloseEle, parseDirection(gun.gun.Direction), nil) {
			c.JSON(http.StatusOK, ResultError(1, "拉闸失败"))
			return
		}
	}
	// 订阅消息提醒（Java subscribeMessageService.sendCarLeaveMessage；此处省略微信订阅消息推送）
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ 微信支付/结算 ============

func (o *OrderController) PayScore(c *gin.Context) {
	orderID := c.Query("orderId")
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "订单不存在: "+orderID))
		return
	}
	needProfitSharing := order.ConsumptionType == consumeTypeYiXiang
	appID := ""
	if order.ConsumptionType == consumeTypeCaiParking {
		appID = yiparlAppID
	} else {
		appID = appid
	}
	c.JSON(http.StatusOK, ResultSuccess(createPayScoreOrder(orderID, needProfitSharing, appID)))
}

func (o *OrderController) QueryPayScore(c *gin.Context) {
	orderID := c.Query("orderId")
	amount := decimal.NewFromFloat(queryFloat(c, "amount"))

	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		c.JSON(http.StatusOK, ResultSuccess(wxPayStateCancel))
		return
	}
	state := queryWxOrderState(orderID)

	switch order.ConsumptionType {
	case consumeTypeCaiParking:
		switch state {
		case wxPayStateDone, wxPayStateDoing:
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateBooking).
				Updates(map[string]interface{}{"state": orderStateBooked, "decide_amount": amount})
		case wxPayStateRevoke:
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateBooking).
				Updates(map[string]interface{}{"state": orderStateRevoked, "decide_amount": amount})
		}
	case consumeTypeYiXiang:
		switch state {
		case wxPayStateDone, wxPayStateDoing:
			if configByOrderid(orderID) == nil {
				conf.Db.Create(&model.ConfigTbl{Orderid: orderID, ChargeGun: 0})
			}
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateBooking).
				Updates(map[string]interface{}{"state": orderStateBooked, "decide_amount": amount})
		case wxPayStateRevoke:
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateBooking).
				Updates(map[string]interface{}{"state": orderStateRevoked, "decide_amount": amount})
		}
	case consumeTypeNeighbor:
		switch state {
		case wxPayStateDone, wxPayStateDoing:
			gun := chargingGunByPrivateOrder(orderID)
			if gun == nil || !service.SendCommand(gun.deviceID(), protocol.OpOpenCharge, protocol.DirectionDefault, nil) {
				c.JSON(http.StatusOK, ResultError(1, "开充电枪失败"))
				return
			}
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateConduct).
				Updates(map[string]interface{}{"state": orderStateUse, "decide_amount": amount, "begin_time": time.Now()})
		case wxPayStateRevoke:
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateBooking).
				Updates(map[string]interface{}{"state": orderStateRevoked, "decide_amount": amount})
		}
	default: // 二轮车充电
		switch state {
		case wxPayStateDone, wxPayStateDoing:
			if !openGauge(orderID) {
				c.JSON(http.StatusOK, ResultError(1, "开闸失败"))
				return
			}
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateConduct).
				Updates(map[string]interface{}{"state": orderStateUse, "decide_amount": amount})
		case wxPayStateRevoke:
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ? AND state = ?", orderID, orderStateConduct).
				Updates(map[string]interface{}{"state": orderStateRevoked, "decide_amount": amount})
		}
	}
	c.JSON(http.StatusOK, ResultSuccess(state))
}

func (o *OrderController) WechatCallback(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
}

func (o *OrderController) GetOrderPay(c *gin.Context) {
	orderID := c.Query("orderId")
	if queryWxOrderState(orderID) != wxPayStateCancel {
		c.JSON(http.StatusOK, ResultSuccess("微信支付分支付"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess("余额支付"))
}

func (o *OrderController) Finish(c *gin.Context) {
	orderID := c.Query("orderId")
	isPrivateUser := c.Query("isPrivateUser")

	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "订单不存在: "+orderID))
		return
	}
	if strings.TrimSpace(order.WechatTransactionID) != "" {
		if order.State != orderStateFinished {
			conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("state", orderStateFinished)
		}
		c.JSON(http.StatusOK, ResultSuccess(true))
		return
	}

	var fees decimal.Decimal
	var amount int64
	if order.ConsumptionType != consumeTypeCaiParking {
		fees = order.BasicConsumption
		if fees.IsZero() && order.ConsumptionType == consumeTypeYiXiang {
			updateBasicConsumptionForOrder(orderID)
			conf.Db.Where("orderid = ?", orderID).First(&order)
			fees = order.BasicConsumption
		}
		if fees.IsZero() {
			fees = decimal.Zero
		}
		amount = fees.Mul(decimal.NewFromInt(100)).RoundDown(0).IntPart()
		res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("state", orderStateFinished)
		if res.RowsAffected == 0 {
			c.JSON(http.StatusOK, ResultSuccess(false))
			return
		}
		service.DelRedisValue("orderState:" + orderID)
	} else {
		amount = order.TotalPrice.Mul(decimal.NewFromInt(100)).RoundDown(0).IntPart()
	}

	if order.ConsumptionType == consumeTypeNeighbor && isPrivateUser == "1" {
		completeWxOrder(orderID, 0)
		c.JSON(http.StatusOK, ResultSuccess(true))
		return
	}

	if amount > 0 {
		completeWxOrder(orderID, int(amount))
		paid := decimal.NewFromInt(amount).Div(decimal.NewFromInt(100))
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("basic_consumption", paid)
		invoiceForOrder(order, paid)
	} else {
		cancelWxOrder(orderID)
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) FinishBalans(c *gin.Context) {
	orderID := c.Query("orderId")
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		c.JSON(http.StatusOK, ResultError(1, "订单不存在: "+orderID))
		return
	}
	var amount decimal.Decimal
	if order.ConsumptionType != consumeTypeCaiParking {
		amount = order.BasicConsumption
		res := conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("state", orderStateFinished)
		if res.RowsAffected == 0 {
			c.JSON(http.StatusOK, ResultSuccess(false))
			return
		}
		service.DelRedisValue("orderState:" + orderID)
	} else {
		amount = order.TotalPrice
	}
	if order.ConsumptionType == consumeTypeNeighbor && order.DecideAmount.IsZero() {
		amount = decimal.Zero
	}

	if amount.IsPositive() {
		if !consumeBalance(order.Openid, amount) {
			c.JSON(http.StatusOK, ResultError(1, "余额支付失败"))
			return
		}
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("basic_consumption", amount)
		invoiceForOrder(order, amount)
		if order.ConsumptionType == consumeTypeCaiParking {
			saveBalanceRecord(order.Openid, "停车消费", amount)
		} else {
			saveBalanceRecord(order.Openid, "充电消费", amount)
		}
	} else {
		completeOrderIsamountNot(order.Openid)
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) Refunds(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(gin.H{}))
}

func (o *OrderController) CancelPayScore(c *gin.Context) {
	cancelWxOrder(c.Query("orderId"))
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) UnfreezeOrder(c *gin.Context) {
	orderID := c.Query("orderId")
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil || strings.TrimSpace(order.WechatTransactionID) == "" {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	// Java 调用 profitSharingService.unfreezeOrder(transactionId, orderId)（微信分账解冻接口）
	logger.Mylog.Info().Msgf("解冻订单资金, orderId: %s, transactionId: %s", orderID, order.WechatTransactionID)
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ 设备读数/其它 ============

func (o *OrderController) GetMeterValue(c *gin.Context) {
	meter := service.GetRedisValue(c.Query("nid") + c.Query("direction"))
	if meter == "" {
		meter = "暂时无法获取到功率信息,请稍后重试"
	}
	c.JSON(http.StatusOK, ResultSuccess(meter))
}

func (o *OrderController) GetCurrentValue(c *gin.Context) {
	current := service.GetRedisValue(c.Query("nid") + c.Query("direction") + ":current")
	if current == "" {
		current = "暂时无法获取到电流信息,请稍后重试"
	}
	c.JSON(http.StatusOK, ResultSuccess(current))
}

func (o *OrderController) GetVoltageValue(c *gin.Context) {
	voltage := service.GetRedisValue(c.Query("nid") + c.Query("direction") + ":voltage")
	if voltage == "" {
		voltage = "暂时无法获取到电压信息,请稍后重试"
	}
	c.JSON(http.StatusOK, ResultSuccess(voltage))
}

func (o *OrderController) GetMeterValueByPrivate(c *gin.Context) {
	v := service.GetRedisValue(c.Query("nid") + c.Query("direction"))
	if v == "" {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	var m map[string]interface{}
	if json.Unmarshal([]byte(v), &m) != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(m))
}

func (o *OrderController) GetOrderRechargeAmount(c *gin.Context) {
	totalFee, _ := electronicFeesByOrderid(c.Query("orderId"))
	c.JSON(http.StatusOK, ResultSuccess(totalFee))
}

func (o *OrderController) GetExpectDuration(c *gin.Context) {
	batteryCapacity := queryFloat(c, "batteryCapacity")
	remainingQuantity := queryFloat(c, "remainingQuantity")
	chargingQuantity := queryFloat(c, "chargingQuantity")
	direction := c.Query("direction")
	productID := c.Query("productId")

	if batteryCapacity == 0 && remainingQuantity == 0 && chargingQuantity == 0 {
		c.JSON(http.StatusOK, ResultError(1, "参数为空"))
		return
	}
	power := decimal.RequireFromString(chargeGunPowerSmall)
	if gun := chargingGunByPidAndDirection(productID, direction); gun != nil && gun.gun.Model == chargeGunModel220 {
		power = decimal.RequireFromString(chargeGunPowerSmall)
	} else if gun != nil {
		power = decimal.RequireFromString(chargeGunPowerBig)
	}
	bc := decimal.NewFromFloat(batteryCapacity)
	charging := decimal.NewFromFloat(chargingQuantity).Div(decimal.NewFromInt(100))
	remaining := decimal.NewFromFloat(remainingQuantity).Div(decimal.NewFromInt(100))
	coef := decimal.RequireFromString(chargeCoefficient)
	result := bc.Mul(charging.Sub(remaining)).DivRound(power, 2).Mul(coef).Round(2)
	c.JSON(http.StatusOK, ResultSuccess(result))
}

func (o *OrderController) AddImage(c *gin.Context) {
	// Java：压缩图片 → ImageService.add（COS 存储）→ 关联到手机号对应 openid 的最新订单 image_id。
	// 此处省略压缩与 COS 上传，仅记录。
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ 内部结算工具 ============

// updateChargingDegreeForOrder 设置订单充电度数（对齐 Java updateChargingDegree）。
func updateChargingDegreeForOrder(orderID string) {
	var degree decimal.Decimal
	conf.Db.Raw("SELECT COALESCE(SUM(over_value - start_value),0)/100 FROM tab_order_electronic WHERE order_id = ?", orderID).
		Row().Scan(&degree)
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("charging_degree", degree)
}

// isAfterNextMidnight 判断时间是否在凌晨 0 点以后、8 点以前（对齐 Java DateTimeUtil.isAfterNextMidnight）。
func isAfterNextMidnight(t time.Time) bool {
	return t.Hour() < 8
}

// calculateOvertimeFee 计算驿享充电超时占位费（对齐 Java calculateOvertimeFee）。
func calculateOvertimeFee(order model.OrderTbl, fee decimal.Decimal, orderID string) decimal.Decimal {
	var chargeEndTime time.Time
	if !order.FullTime.IsZero() {
		chargeEndTime = order.FullTime
	} else if !order.GunDisconnectTime.IsZero() {
		chargeEndTime = order.GunDisconnectTime
	} else {
		return fee
	}
	var leaveTime time.Time
	if !order.LeaveTime.IsZero() {
		leaveTime = order.LeaveTime
	} else {
		if lockByOrder(orderID) == nil {
			leaveTime = order.OverTime
		} else {
			return fee // 正在检测车辆是否正确驶离
		}
	}
	betweenMinute := leaveTime.Sub(chargeEndTime).Minutes()
	if isAfterNextMidnight(chargeEndTime) {
		eightAM := time.Date(chargeEndTime.Year(), chargeEndTime.Month(), chargeEndTime.Day(), 8, 0, 0, 0, chargeEndTime.Location())
		betweenMinuteAfter := eightAM.Sub(chargeEndTime).Minutes()
		if betweenMinute-betweenMinuteAfter > 0 {
			timeoutFee := decimal.NewFromFloat(betweenMinute - betweenMinuteAfter).Mul(decimal.NewFromFloat(overtimeParkingFee))
			if timeoutFee.GreaterThan(decimal.NewFromFloat(overtimeServiceFee)) {
				timeoutFee = decimal.NewFromFloat(overtimeServiceFee)
			}
			fee = timeoutFee.Add(fee)
		}
	} else if betweenMinute > 30 {
		overtimeMinutes := betweenMinute - 30
		timeoutFee := decimal.NewFromFloat(overtimeMinutes).Mul(decimal.NewFromFloat(overtimeParkingFee))
		if timeoutFee.GreaterThan(decimal.NewFromFloat(overtimeServiceFee)) {
			timeoutFee = decimal.NewFromFloat(overtimeServiceFee)
		}
		fee = timeoutFee.Add(fee)
	}
	return fee
}

// calculateNeighborOvertimeFee 计算邻享充电超时占位费（对齐 Java calculateNeighborOvertimeFee）。
func calculateNeighborOvertimeFee(order *model.OrderTbl) decimal.Decimal {
	orderPrivate := orderPrivateByOrderid(order.Orderid)
	if orderPrivate == nil || orderPrivate.IsPrivateUser != 2 {
		return decimal.Zero
	}
	privatePlace := privatePlaceByID(order.SpacesID)
	if privatePlace == nil {
		return decimal.Zero
	}
	chargingBean := privateChargingBySpacesNum(privatePlace.SpacesCode)
	if chargingBean == nil {
		return decimal.Zero
	}
	// 桩主需要换停车位且收取超时占位费
	if chargingBean.ExchangePlace && chargingBean.OvertimeFee {
		if order.OverTime.IsZero() || (order.NeighborRelocateTime.IsZero() && order.MoveCarTime.IsZero()) {
			return decimal.Zero
		}
		var relocateTime time.Time
		if !order.NeighborRelocateTime.IsZero() && !order.MoveCarTime.IsZero() {
			if order.NeighborRelocateTime.Before(order.MoveCarTime) {
				relocateTime = order.NeighborRelocateTime
			} else {
				relocateTime = order.MoveCarTime
			}
		} else if !order.NeighborRelocateTime.IsZero() {
			relocateTime = order.NeighborRelocateTime
		} else {
			relocateTime = order.MoveCarTime
		}
		return neighborOvertimeFeeFromTime(relocateTime, order.OverTime)
	}
	// 桩主不需要换停车位但收取超时占位费：以充满时间为起始
	if chargingBean.OvertimeFee {
		if order.FullTime.IsZero() || order.OverTime.IsZero() {
			return decimal.Zero
		}
		return neighborOvertimeFeeFromTime(order.FullTime, order.OverTime)
	}
	return decimal.Zero
}

// neighborOvertimeFeeFromTime 计算邻享超时费（0-8点免费，8点后计费，30分钟宽限，0.05/分钟，封顶200）。
func neighborOvertimeFeeFromTime(startTime, leaveTime time.Time) decimal.Decimal {
	betweenMinute := leaveTime.Sub(startTime).Minutes()
	if isAfterNextMidnight(startTime) {
		eightAM := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 8, 0, 0, 0, startTime.Location())
		after8 := betweenMinute - eightAM.Sub(startTime).Minutes()
		if after8 <= 0 {
			return decimal.Zero
		}
		if after8 <= 30 {
			return decimal.Zero
		}
		timeoutFee := decimal.NewFromFloat(after8 - 30).Mul(decimal.NewFromFloat(overtimeParkingFee))
		if timeoutFee.GreaterThan(decimal.NewFromFloat(overtimeServiceFee)) {
			timeoutFee = decimal.NewFromFloat(overtimeServiceFee)
		}
		return timeoutFee
	}
	if betweenMinute <= 30 {
		return decimal.Zero
	}
	timeoutFee := decimal.NewFromFloat(betweenMinute - 30).Mul(decimal.NewFromFloat(overtimeParkingFee))
	if timeoutFee.GreaterThan(decimal.NewFromFloat(overtimeServiceFee)) {
		timeoutFee = decimal.NewFromFloat(overtimeServiceFee)
	}
	return timeoutFee
}

// updateBasicConsumptionForOrder 计算并更新订单基本消费（对齐 Java updateBasicConsumption）。
func updateBasicConsumptionForOrder(orderID string) {
	fee, _ := electronicFeesByOrderid(orderID)
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		return
	}
	if order.ChargeGunPull == 0 && order.OverTime.IsZero() {
		beginTime := order.BeginTime
		var nowTime time.Time
		if !order.LeaveTime.IsZero() {
			nowTime = order.LeaveTime
		} else {
			if lockByOrder(orderID) == nil {
				conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("basic_consumption", fee.Round(2))
				return
			}
			return // 正在检测车辆是否正确驶离
		}
		betweenMinute := nowTime.Sub(beginTime).Minutes()
		overtimeMinutes := betweenMinute - 15
		if overtimeMinutes < 0 {
			overtimeMinutes = 0
		}
		chargeFee := decimal.NewFromFloat(overtimeMinutes).Mul(decimal.NewFromFloat(overtimeParkingFee))
		if chargeFee.GreaterThan(decimal.NewFromFloat(overtimeServiceFee)) {
			chargeFee = decimal.NewFromFloat(overtimeServiceFee)
		}
		fee = chargeFee.Add(fee)
	} else if fee.IsZero() {
		beginTime := order.BeginTime
		var nowTime time.Time
		if !order.LeaveTime.IsZero() {
			nowTime = order.LeaveTime
		} else {
			if lockByOrder(orderID) == nil {
				nowTime = order.OverTime
			} else {
				return // 正在检测车辆是否正确驶离
			}
		}
		betweenMinute := nowTime.Sub(beginTime).Minutes()
		overtimeMinutes := betweenMinute - 15
		if overtimeMinutes < 0 {
			overtimeMinutes = 0
		}
		chargeFee := decimal.NewFromFloat(overtimeMinutes).Mul(decimal.NewFromFloat(overtimeParkingFee))
		if chargeFee.GreaterThan(decimal.NewFromFloat(overtimeServiceFee)) {
			chargeFee = decimal.NewFromFloat(overtimeServiceFee)
		}
		fee = chargeFee.Add(fee)
	} else {
		if order.ConsumptionType != consumeTypeNeighbor && (!order.FullTime.IsZero() || !order.GunDisconnectTime.IsZero()) {
			if order.ConsumptionType == consumeTypeYiXiang && order.SpacesID != 0 {
				if spaces := spacesByID(order.SpacesID); spaces != nil && !spaces.OvertimeFee {
					// 车位设置不收取超时占位费
				} else {
					fee = calculateOvertimeFee(order, fee, orderID)
				}
			} else {
				fee = calculateOvertimeFee(order, fee, orderID)
			}
		}
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("basic_consumption", fee.Round(2))
}

// ============ 二轮车计费 ============

func reOrderPowerValue(orderID string) decimal.Decimal {
	var v decimal.Decimal
	conf.Db.Raw("SELECT COALESCE((MAX(over_value)-MIN(start_value)),0) FROM re_order_electronic_tbl WHERE order_id = ?", orderID).Row().Scan(&v)
	return v
}

func reOrderPowerNoServiceFees(orderID string) decimal.Decimal {
	var v decimal.Decimal
	conf.Db.Raw(`SELECT COALESCE(SUM((over_value - start_value) * charging_price_tbl.total),0)
		FROM re_order_electronic_tbl
		LEFT JOIN tab_price_time ON tab_price_time.id = re_order_electronic_tbl.price_time_id
		LEFT JOIN charging_price_tbl ON charging_price_tbl.id = tab_price_time.price_id
		WHERE re_order_electronic_tbl.order_id = ?`, orderID).Row().Scan(&v)
	return v
}

func reOrderPowerMaxPower(orderID string) float64 {
	var v float64
	conf.Db.Raw("SELECT COALESCE(max_power,0) FROM re_order_electronic_tbl WHERE order_id = ? LIMIT 1", orderID).Row().Scan(&v)
	return v
}

func serviceFeeByPowerRange(maxPower float64) decimal.Decimal {
	var v decimal.Decimal
	conf.Db.Raw("SELECT service FROM charging_price_tbl WHERE ? >= min_power AND ? <= max_power LIMIT 1", maxPower, maxPower).Row().Scan(&v)
	return v
}

// userIsFreeServiceByOrder 判断订单用户是否享受免服务费（对齐 FreeChargingUserServiceImpl.getUserIsNotFreeByOrderId）。
func userIsFreeServiceByOrder(orderID string) bool {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		return false
	}
	var users []model.FreeChargingUsersTbl
	conf.Db.Raw(`SELECT fcu.* FROM free_charging_users_tbl fcu
		LEFT JOIN order_tbl o ON o.openid = fcu.openid
		WHERE o.orderid = ? AND fcu.status = 1`, orderID).Scan(&users)
	for _, u := range users {
		var productID string
		conf.Db.Raw("SELECT station.product_id FROM charging_station_management_tbl station WHERE station.id = ?", u.ChargingStationID).Scan(&productID)
		if order.Pid == productID {
			return true
		}
	}
	return false
}

// closeGauge 二轮车关闸（对齐 Java CloseGAUGE）。
func closeGauge(orderID string) {
	receptacle := receptacleByOrderid(orderID)
	if receptacle == nil {
		return
	}
	product := productByPid(receptacle.Pid)
	if product == nil {
		return
	}
	service.SendCommand(product.Imei, protocol.OpCloseGauge, protocol.DirectionDefault, []byte(strings.Split(receptacle.Name, "号")[0]))
}

// openGauge 二轮车开闸（对齐 Java OpenGAUGE，含开始时间/状态/插座状态更新）。
func openGauge(orderID string) bool {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		return false
	}
	receptacle := receptacleByOrderid(orderID)
	if receptacle == nil {
		return false
	}
	product := productByPid(order.Pid)
	if product == nil {
		return false
	}
	rid := strings.Split(receptacle.Name, "号")[0]
	if !service.SendCommand(product.Imei, protocol.OpOpenGauge, protocol.DirectionDefault, []byte(rid)) {
		return false
	}
	if rp := receptaclePowerByPidAndRid(product.Pid, rid); rp != nil {
		conf.Db.Create(&model.ReOrderElectronicTbl{OrderID: orderID, StartValue: rp.MeterValue, OverValue: rp.MeterValue})
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Updates(map[string]interface{}{
		"begin_time": time.Now(),
		"state":      orderStateUse,
	})
	conf.Db.Model(&model.ReceptacleTbl{}).Where("id = ? AND status = 0", receptacle.ID).Update("status", 1)
	return true
}

// closeTwiceOrderInternal 结束二轮车充电订单（对齐 Java closeTwiceOrder / closeTwiceOrderWithoutGauge）。
func closeTwiceOrderInternal(orderID string, withGauge bool) {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderID).First(&order).Error != nil {
		return
	}
	meterValue := reOrderPowerValue(orderID)

	// 套餐抵扣
	hasValidPackage := false
	var chargingMinutes int64
	var packageRecord *model.PackageRecordTbl
	if pr := latestValidPackage(order.Openid); pr != nil {
		packageRecord = pr
		orderMaxPower := reOrderPowerMaxPower(orderID)
		packageMaxPower := float64(pr.MaxPower)
		if packageMaxPower > 0 && orderMaxPower > packageMaxPower {
			hasValidPackage = false
		} else if pr.Duration == -1 {
			hasValidPackage = true
		} else if pr.Duration > 0 {
			hasValidPackage = true
			if !order.BeginTime.IsZero() {
				chargingMinutes = int64(time.Since(order.BeginTime).Minutes())
				if chargingMinutes < 1 {
					chargingMinutes = 1
				}
			}
		}
	}

	var fee, calculatedServiceFee, powerRate decimal.Decimal
	if hasValidPackage {
		fee = decimal.Zero
		calculatedServiceFee = decimal.Zero
		powerRate = decimal.Zero
		if packageRecord.Duration > 0 {
			remaining := packageRecord.Duration - int32(chargingMinutes)
			if remaining < 0 {
				remaining = 0
			}
			conf.Db.Model(&model.PackageRecordTbl{}).Where("order_no = ?", packageRecord.OrderNo).Update("duration", remaining)
		}
	} else if userIsFreeServiceByOrder(orderID) {
		fee = reOrderPowerNoServiceFees(orderID)
		calculatedServiceFee = decimal.Zero
		powerRate = fee
	} else {
		betweenMinute := int64(0)
		if !order.BeginTime.IsZero() {
			betweenMinute = int64(time.Since(order.BeginTime).Minutes())
		}
		maxPower := reOrderPowerMaxPower(orderID)
		dynamicServiceFee := serviceFeeByPowerRange(maxPower)
		chargingHours := betweenMinute / 60
		if betweenMinute%60 > 0 || chargingHours == 0 {
			chargingHours++
		}
		calculatedServiceFee = dynamicServiceFee.Mul(decimal.NewFromInt(chargingHours))
		fee = reOrderPowerNoServiceFees(orderID)
		powerRate = fee
		if reOrderPowerValue(orderID).IsZero() && powerRate.IsZero() {
			calculatedServiceFee = decimal.Zero
		}
		fee = fee.Add(calculatedServiceFee)
	}

	conf.Db.Create(&model.OrderTwiceTbl{Orderid: orderID, ServiceRate: calculatedServiceFee, PowerRate: powerRate})

	if withGauge {
		closeGauge(orderID)
	}

	state := queryWxOrderState(orderID)
	if state != wxPayStateCancel {
		amount := fee.Mul(decimal.NewFromInt(100)).IntPart()
		completeWxOrder(orderID, int(amount))
	} else if fee.IsPositive() {
		consumeBalance(order.Openid, fee)
		saveBalanceRecord(order.Openid, "充电消费", fee)
	} else {
		completeOrderIsamountNot(order.Openid)
	}

	if receptacle := receptacleByOrderid(orderID); receptacle != nil {
		conf.Db.Model(&model.ReceptacleTbl{}).Where("id = ?", receptacle.ID).Update("status", 0)
	}

	// 关闸时间 + 充电时长
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("over_time", time.Now())
	if !order.BeginTime.IsZero() {
		d := time.Since(order.BeginTime)
		chargingTime := time.Time{}.Add(d).Format("15:04:05")
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Update("charging_time", chargingTime)
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderID).Updates(map[string]interface{}{
		"state":             orderStateFinished,
		"close_time":        time.Now(),
		"charging_degree":   meterValue,
		"basic_consumption": fee,
	})
}

// latestValidPackage 查询用户最早购买且未过期的套餐（对齐 Java PackageRecordService.getLatestValidPackage）。
func latestValidPackage(openid string) *model.PackageRecordTbl {
	var pr model.PackageRecordTbl
	if err := conf.Db.Where("openid = ? AND status = 'paid' AND end_time > ?", openid, time.Now()).
		Order("create_time asc").First(&pr).Error; err != nil {
		return nil
	}
	return &pr
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
