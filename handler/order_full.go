package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/protocol"
	"caicai-go/repository"
	"caicai-go/service"
)

// newOrderID 生成订单号。
func newOrderID() string {
	return "OD" + time.Now().Format("20060102150405") + strconv.Itoa(int(time.Now().UnixNano()%100000))
}

// orderDeviceID 根据订单解析设备标识（产品 IMEI 或 NID）。用于二轮车等 pid 直接关联产品的订单。
func orderDeviceID(orderID string) string {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || order.Pid == "" {
		return ""
	}
	prod := repository.ProductByPid(order.Pid)
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
	return repository.FirstUnfinishedOrderByOpenid(openid)
}

// getByOpenidNotFinish 按 openid 查询未完成订单（对齐 Java getByOpenidNotFinish）。
func getByOpenidNotFinish(openid string) *model.OrderTbl {
	return getNotFinishOrder(openid)
}

// getNotFinishBySpacesIdList 根据车位查询未完成订单（含一桩一锁关联车位），对齐 Java getNotFinishBySpacesId。
func getNotFinishBySpacesIdList(spacesID int32) []model.OrderTbl {
	list := repository.UnfinishedNonNeighborOrdersBySpacesID(spacesID)

	spaces := repository.SpacesByID(spacesID)
	if spaces == nil || spaces.LockID == "" {
		return list
	}
	if !repository.IsOneLockPerSpot(spacesID) {
		return list
	}
	otherID := repository.OtherSpacesIDBySpacesID(spacesID)
	if otherID == 0 {
		return list
	}
	list1 := repository.UnfinishedOrdersBySpacesID(otherID)
	return append(list, list1...)
}

// ============ 查询 ============

func (o *OrderController) GetByOrderId(c *gin.Context) {
	orderID := c.Query("orderId")
	if len(orderID) < 10 {
		c.JSON(http.StatusOK, ResultError(1, "订单id为空"))
		return
	}
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderToDTO(*order)))
}

func (o *OrderController) GetAllOrders(c *gin.Context) {
	orders := repository.LatestOrdersByOpenid(c.Query("openid"), 20)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

func (o *OrderController) GetOrdersByPage(c *gin.Context) {
	current, size := parsePage(c)
	openid := c.Query("openid")
	total, orders := repository.PageOrdersByOpenid(openid, pageOffset(current, size), size)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (o *OrderController) GetOngoingOrders(c *gin.Context) {
	orders := repository.OngoingOrdersByOpenid(c.Query("openid"))
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

func (o *OrderController) GetDoneOrders(c *gin.Context) {
	orders := repository.OrdersByOpenidAndState(c.Query("openid"), repository.OrderStateFinished)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

func (o *OrderController) GetWaitPayOrders(c *gin.Context) {
	orders := repository.OrdersByOpenidAndState(c.Query("openid"), repository.OrderStateWaitPay)
	list := make([]OrderDTO, 0, len(orders))
	for _, od := range orders {
		list = append(list, orderToDTO(od))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

func (o *OrderController) GetOrdersByPageForCaiCai(c *gin.Context) {
	current, size := parsePage(c)
	openid := c.Query("openid")
	total, orders := repository.PageOrdersByOpenidAndConsumption(openid, repository.ConsumeTypeCaiParking, pageOffset(current, size), size)
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
	order, err := repository.WaitReturnGunOrderByPrivatePlaceID(c.Query("spacesId"))
	if err != nil || order == nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(orderToDTO(*order)))
}

func (o *OrderController) GetOrderState(c *gin.Context) {
	state := service.GetRedisValue("orderState:" + c.Query("orderId"))
	if state == "" {
		state = "暂时无法获取到订单信息,请稍后重试"
	}
	c.JSON(http.StatusOK, ResultSuccess(state))
}

func (o *OrderController) GetplaceUsable(c *gin.Context) {
	orders := repository.ActiveCaiParkingOrdersByLockid(c.Query("lockId"))
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
		State:       repository.OrderStateBooking,
		ReserveTime: timePtr(time.Now()),
	}
	if err := repository.CreateOrder(&order); err != nil {
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

	spaces := repository.SpacesByLockID(lockid)
	var spacesID int32
	if spaces != nil {
		spacesID = spaces.ID
	}
	place := repository.PlaceByLockid(lockid)
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
		ReserveTime:     timePtr(time.Now()),
		State:           repository.OrderStateConduct,
		PlateNum:        c.Query("plateNum"),
		ConsumptionType: repository.ConsumeTypeCaiParking,
		GiftAmount:      "暂未开放~",
		StartMode:       "扫码",
		BeginTime:       timePtr(time.Now()),
		ChargingRates:   rate,
	}
	if err := repository.CreateOrder(&order); err != nil {
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByCharge(c *gin.Context) {
	openid := c.GetString("openid")
	pid := c.Query("pid")
	direction := c.Query("direction")

	spaces := repository.SpacesByPidAndDirection(pid, direction)
	if spaces == nil {
		c.JSON(http.StatusOK, ResultError(1, "车位不存在"))
		return
	}
	if getNotFinishOrder(openid) != nil {
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}
	var totalPrice decimal.Decimal
	if gun := repository.ChargingGunByID(spaces.ChargingGunID); gun != nil {
		totalPrice = gun.UnitPrice
	}

	order := model.OrderTbl{
		Orderid:     newOrderID(),
		TotalPrice:  totalPrice,
		Openid:      openid,
		SpacesID:    spaces.ID,
		ReserveTime: timePtr(time.Now()),
		State:       repository.OrderStateBooking,
	}
	if err := repository.CreateOrder(&order); err != nil {
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	repository.CreateConfig(order.Orderid)
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByTwiceCharge(c *gin.Context) {
	openid := c.GetString("openid")
	pid := c.Query("pid")
	chargeType := c.Query("ChargeType")
	receptacleID := int32(queryInt(c, "ReceptacleId", 0))
	chargeTime := c.Query("ChargeTime")

	station, err := repository.ChargingStationByPid(pid)
	if err != nil {
		c.JSON(http.StatusOK, ResultError(1, "充电站不存在"))
		return
	}
	receptacle := repository.ReceptacleByID(receptacleID)

	notFinish := getNotFinishOrder(openid)
	if notFinish != nil {
		if notFinish.State == repository.OrderStateConduct {
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
		ReserveTime:         timePtr(time.Now()),
		ConsumptionLocation: station.Name,
		State:               repository.OrderStateConduct,
		ConsumptionType:     repository.ConsumeTypeTwice,
		GiftAmount:          "暂未开放~",
		StartMode:           "扫码",
		StopMode:            "扫码自停",
		Lockid:              lockid,
		ChargingRates:       decimal.RequireFromString(repository.OrderTwiceChargeRate),
		Pid:                 pid,
	}
	if chargeType == "TimeSelected" {
		order.TwiceChargeTime = chargeTime
	}
	if err := repository.CreateOrder(&order); err != nil {
		logger.Mylog.Err(err).Msg("预约失败")
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) AddByPrivateCharge(c *gin.Context) {
	openid := c.GetString("openid")
	privatePlaceID := int32(queryInt(c, "privatePlaceId", 0))

	privatePlace := repository.PrivatePlaceByID(privatePlaceID)
	if privatePlace == nil {
		c.JSON(http.StatusOK, ResultError(1, "车位不存在"))
		return
	}
	if getNotFinishOrder(openid) != nil {
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}

	if existing, err := repository.UnfinishedNeighborOrderBySpacesID(privatePlaceID); err == nil {
		switch existing.State {
		case repository.OrderStateConduct:
			c.JSON(http.StatusOK, ResultError(1, "前序订单正在进行中,请等待"))
			return
		case repository.OrderStateUse:
			c.JSON(http.StatusOK, ResultError(1, "前序订单正在使用中,请等待"))
			return
		case repository.OrderStateWaitGun:
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
		ReserveTime:         timePtr(time.Now()),
		ConsumptionLocation: privatePlace.Place,
		ConsumptionType:     repository.ConsumeTypeNeighbor,
		GiftAmount:          "暂未开放~",
		StartMode:           "扫码",
		StopMode:            "扫码自停",
		State:               repository.OrderStateConduct,
	}
	if err := repository.CreateOrder(&order); err != nil {
		c.JSON(http.StatusOK, ResultError(1, "生成订单失败"))
		return
	}
	repository.InsertOrderPrivate(&model.OrderPrivateTbl{Orderid: order.Orderid, IsPrivateUser: int32(queryInt(c, "isPrivateUser", 0))})
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) Booking(c *gin.Context) {
	openid := c.GetString("openid")
	spacesID := int32(queryInt(c, "spacesId", 0))
	startTime := parseTimePtr(c.Query("startTime"))
	endTime := parseTimePtr(c.Query("endTime"))

	parkingSpaces := repository.SpacesByID(spacesID)
	if getNotFinishOrder(openid) != nil {
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}
	if du := repository.DriverUseByOpenid(openid); du == nil || du.ReserveNumber == 0 {
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
		ReserveTime:      timePtr(time.Now()),
		State:            repository.OrderStateBooking,
		BookingStartTime: timePtr(startTime),
		BookingEndTime:   timePtr(endTime),
		ConsumptionType:  repository.ConsumeTypeYiXiang,
		ChargingRates:    repository.CurrentChargingRate(),
		StartMode:        "扫码",
		StopMode:         "扫码自停",
		GiftAmount:       "暂未开放~",
	}
	if parkingSpaces != nil {
		order.ConsumptionLocation = parkingSpaces.Place
	}
	if err := repository.CreateOrder(&order); err != nil {
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

	parkingSpaces := repository.SpacesByID(spacesID)
	if getNotFinishOrder(openid) != nil {
		c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
		return
	}
	if du := repository.DriverUseByOpenid(openid); du == nil || du.ReserveNumber == 0 {
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
		if gun := repository.ChargingGunByID(parkingSpaces.ChargingGunID); gun != nil {
			if repository.ChargingStateByProductAndDirection(gun.ProductID, gun.Direction) != nil {
				c.JSON(http.StatusOK, ResultError(1, "当前充电桩紧急按钮被按下,暂不可用"))
				return
			}
		}
	}

	order := model.OrderTbl{
		Orderid:          newOrderID(),
		Openid:           openid,
		SpacesID:         spacesID,
		ReserveTime:      timePtr(time.Now()),
		State:            repository.OrderStateBooking,
		BookingStartTime: timePtr(startTime),
		BookingEndTime:   timePtr(endTime),
		ConsumptionType:  repository.ConsumeTypeYiXiang,
		ChargingRates:    repository.CurrentChargingRate(),
		StartMode:        "扫码",
		StopMode:         "扫码自停",
		GiftAmount:       "暂未开放~",
	}
	if parkingSpaces != nil {
		order.ConsumptionLocation = parkingSpaces.Place
	}
	if err := repository.CreateOrder(&order); err != nil {
		c.JSON(http.StatusOK, ResultError(1, "预约失败"))
		return
	}
	// 时间模式：预约结束后自动断电（Java 通过 RabbitMQ 定时任务，此处省略定时调度）
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

func (o *OrderController) NewBookingByYiXiang(c *gin.Context) {
	openid := c.GetString("openid")
	spacesID := int32(queryInt(c, "spacesId", 0))

	parkingSpaces := repository.SpacesByID(spacesID)
	notFinish := getNotFinishOrder(openid)
	if notFinish != nil {
		if notFinish.State == repository.OrderStateBooked || notFinish.State == repository.OrderStateBooking {
			cancelOrderInternal(notFinish.Orderid, "0")
		} else {
			c.JSON(http.StatusOK, ResultError(1, "已有未完成订单"))
			return
		}
	}
	if parkingSpaces != nil && parkingSpaces.ChargingGunID != 0 {
		if gun := repository.ChargingGunByID(parkingSpaces.ChargingGunID); gun != nil {
			if repository.ChargingStateByProductAndDirection(gun.ProductID, gun.Direction) != nil {
				c.JSON(http.StatusOK, ResultError(1, "当前充电桩紧急按钮被按下,暂不可用"))
				return
			}
		}
	}

	order := model.OrderTbl{
		Orderid:         newOrderID(),
		Openid:          openid,
		SpacesID:        spacesID,
		ReserveTime:     timePtr(time.Now()),
		State:           repository.OrderStateBooking,
		ConsumptionType: repository.ConsumeTypeYiXiang,
		ChargingRates:   repository.CurrentChargingRate(),
		StartMode:       "扫码",
		StopMode:        "扫码自停",
		GiftAmount:      "暂未开放~",
	}
	if parkingSpaces != nil {
		order.ConsumptionLocation = parkingSpaces.Place
	}
	if err := repository.CreateOrder(&order); err != nil {
		c.JSON(http.StatusOK, ResultError(1, "创建订单失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(order.Orderid))
}

// ============ 取消 / 结算 / 控制 ============

// cancelOrderInternal 取消订单（内部调用，对齐 Java OrderServiceImpl.cancelOrder）。
func cancelOrderInternal(orderID, price string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
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
		repository.CompleteOrderIsamountNotNull(order.Openid, price)
		p := price
		if p == "undefined" || p == "null" {
			p = "0"
		}
		if pv, err := strconv.Atoi(p); err == nil && pv > 0 {
			repository.SaveBalanceRecord(order.Openid, "超时取消扣除手续费", decimal.NewFromInt(int64(pv)))
		}
	}

	repository.UpdateOrderFields(orderID, map[string]interface{}{
		"close_time":        time.Now(),
		"basic_consumption": decimal.Zero,
		"charging_degree":   decimal.Zero,
	})

	if order.ConsumptionType != repository.ConsumeTypeTwice && order.ConsumptionType != repository.ConsumeTypeNeighbor {
		if spaces := repository.SpacesByID(order.SpacesID); spaces != nil && spaces.SpacesCode != "" {
			repository.UpdatePlaceState(spaces.SpacesCode, "可使用")
		}
	}

	if du := repository.DriverUseByOpenid(order.Openid); du != nil && du.ReserveNumber > 0 {
		repository.UpdateDriverUseReserveNumber(order.Openid, du.ReserveNumber-1)
	}

	return repository.UpdateOrderState(orderID, repository.OrderStateRevoked) > 0
}

func (o *OrderController) CancelOrder(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(cancelOrderInternal(c.Query("orderId"), c.Query("price"))))
}

func (o *OrderController) Close(c *gin.Context) {
	orderID := c.Query("orderId")
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultError(1, "查无此订单!"))
		return
	}
	if order.State == repository.OrderStateFinished {
		c.JSON(http.StatusOK, ResultError(1, "此订单已完成!"))
		return
	}
	if order.ConsumptionType == repository.ConsumeTypeNeighbor {
		linXiangCloseOrder(order)
	} else {
		normalCloseOrder(order)
	}
	res := repository.UpdateOrderFields(orderID, map[string]interface{}{
		"close_time": time.Now(),
		"state":      repository.OrderStateWaitPay,
	})
	c.JSON(http.StatusOK, ResultSuccess(res > 0))
}

// normalCloseOrder 结束普通订单（非邻享充电），对齐 Java NormalCloseOrder。
func normalCloseOrder(order *model.OrderTbl) {
	gun := repository.ChargingGunByOrder(order.Orderid)
	if gun != nil {
		if isZeroTime(order.OverTime) && order.ChargingDegree.IsZero() {
			if !service.SendCommand(gun.DeviceID(), protocol.OpCloseEle, -1) {
				// Java 抛「拉闸失败」；这里保守继续
				logger.Mylog.Warn().Msgf("拉闸失败, orderId: %s", order.Orderid)
			}
		}
	}
	if order.ChargeGunPull == 1 && order.ChargingDegree.IsZero() {
		// Java 抛「请先归还充电枪」；保守继续
		logger.Mylog.Warn().Msgf("请先归还充电枪, 后结束订单~ orderId: %s", order.Orderid)
	}
	repository.UpdateChargingDegreeForOrder(order.Orderid)
	updateBasicConsumptionForOrder(order.Orderid)

	if lock := repository.LockByOrder(order.Orderid); lock != nil {
		service.SendCommand(lock.DeviceID(), protocol.OpClosePlaceLock, -1)
	}
}

// linXiangCloseOrder 结束邻享充电订单，对齐 Java LinXiangCloseOrder。
func linXiangCloseOrder(order *model.OrderTbl) {
	gun := repository.ChargingGunByPrivateOrder(order.Orderid)
	if gun != nil {
		if isZeroTime(order.OverTime) && order.ChargingDegree.IsZero() {
			service.SendCommand(gun.DeviceID(), protocol.OpCloseEle, -1)
		}
	}
	if order.ChargeGunPull == 1 && order.ChargingDegree.IsZero() {
		logger.Mylog.Warn().Msgf("请先归还充电枪后, 再结束订单~ orderId: %s", order.Orderid)
	}
	repository.UpdateChargingDegreeForOrder(order.Orderid)

	orderPrivate := repository.OrderPrivateByOrderid(order.Orderid)
	if orderPrivate == nil {
		return
	}
	var fee decimal.Decimal
	switch orderPrivate.IsPrivateUser {
	case 1: // 桩主：仅电费（total）
		fee = repository.PrivateFeesByPrivateOrderid(order.Orderid)
	case 0: // 毗邻邻居：电费 + 私桩服务费率
		var rate decimal.Decimal
		if privatePlace := repository.PrivatePlaceByID(order.SpacesID); privatePlace != nil {
			if chargingBean := repository.PrivateChargingBySpacesNum(privatePlace.SpacesCode); chargingBean != nil {
				rate = chargingBean.Fee
			}
		}
		fee = repository.PrivateFeesByOrderidAndPrivateUser(order.Orderid, rate)
	case 2: // 普通用户：电费 + 服务费 + 超时占位费
		totalFee, serviceFee := repository.PrivateElectronicByOrderidAndPrivateUser(order.Orderid)
		baseFee := totalFee.Add(serviceFee)
		neighborOvertimeFee := calculateNeighborOvertimeFee(order)
		fee = baseFee.Add(neighborOvertimeFee)
	}
	repository.UpdateOrderBasicConsumption(order.Orderid, fee.Round(2))
}

func (o *OrderController) UpdateOrderState(c *gin.Context) {
	orderID := c.Query("orderId")
	openid := c.Query("openid")
	amount := decimal.NewFromFloat(queryFloat(c, "amount"))

	if !repository.FreezeBalance(openid, amount) {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}

	switch order.ConsumptionType {
	case repository.ConsumeTypeCaiParking:
		if order.State == repository.OrderStateConduct {
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateConduct,
				map[string]interface{}{"state": repository.OrderStateBooked, "decide_amount": amount})
		} else {
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateBooking,
				map[string]interface{}{"state": repository.OrderStateBooked, "decide_amount": amount})
		}
		c.JSON(http.StatusOK, ResultSuccess(true))
	case repository.ConsumeTypeYiXiang:
		if repository.ConfigByOrderid(orderID) == nil {
			repository.CreateConfig(orderID)
		}
		repository.UpdateOrderFieldsByStates(orderID, []string{repository.OrderStateBooking, repository.OrderStateWaitUse},
			map[string]interface{}{"state": repository.OrderStateBooked, "decide_amount": amount})
		c.JSON(http.StatusOK, ResultSuccess(true))
	case repository.ConsumeTypeNeighbor:
		gun := repository.ChargingGunByPrivateOrder(orderID)
		if gun == nil || !service.OpenGunLock(gun.DeviceID()) {
			c.JSON(http.StatusOK, ResultError(1, "开充电枪失败"))
			return
		}
		repository.UpdateOrderFieldsByState(orderID, repository.OrderStateConduct,
			map[string]interface{}{"state": repository.OrderStateUse, "decide_amount": amount, "begin_time": time.Now()})
		c.JSON(http.StatusOK, ResultSuccess(true))
	default: // 二轮车充电
		if !openGauge(orderID) {
			c.JSON(http.StatusOK, ResultError(1, "开闸失败"))
			return
		}
		repository.UpdateOrderFields(orderID, map[string]interface{}{"state": repository.OrderStateUse, "decide_amount": amount})
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
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultError(1, "查无此订单"))
		return
	}
	if !isZeroTime(order.BookingStartTime) {
		if order.BookingStartTime.Sub(time.Now()) > 30*time.Minute {
			c.JSON(http.StatusOK, ResultError(1, "还没到您所预约的充电时间,我们支持提前半小时开始充电服务,请耐心等待!"))
			return
		}
	}
	spaces := repository.SpacesByID(order.SpacesID)
	if spaces == nil {
		c.JSON(http.StatusOK, ResultError(1, "车位不存在"))
		return
	}
	if spaces.ChargingGunID != 0 {
		if gun := repository.ChargingGunByID(spaces.ChargingGunID); gun != nil {
			if repository.ChargingStateByProductAndDirection(gun.ProductID, gun.Direction) != nil {
				c.JSON(http.StatusOK, ResultError(1, "当前充电桩紧急按钮被按下,暂不可用"))
				return
			}
		}
	}

	if spaces.LockID == "" {
		gun := repository.ChargingGunByOrder(orderID)
		if gun == nil {
			c.JSON(http.StatusOK, ResultError(1, "未找到充电枪信息,请更换设备使用"))
			return
		}
		if !service.SendCommand(gun.DeviceID(), protocol.OpOpenCharge, -1) {
			c.JSON(http.StatusOK, ResultError(1, "解锁充电枪失败!"))
			return
		}
		service.SetRedisValue("orderState:"+orderID, "拔枪出桩")
	} else {
		if repository.IsOneLockPerSpot(order.SpacesID) {
			otherID := repository.OtherSpacesIDBySpacesID(order.SpacesID)
			if repository.CountOrdersBySpacesIDState(otherID, repository.OrderStateUse) > 0 {
				c.JSON(http.StatusOK, ResultError(1, "当前车位车辆还未驶离,请稍后再试!"))
				return
			}
		}
		if lock := repository.LockByOrder(orderID); lock != nil {
			if !service.SendCommand(lock.DeviceID(), protocol.OpOpenPlaceLock, -1) {
				c.JSON(http.StatusOK, ResultError(1, "开车位锁失败"))
				return
			}
		}
		service.SetRedisValue("orderState:"+orderID, "驱车入位")
	}

	res := repository.UpdateOrderFields(orderID, map[string]interface{}{"state": repository.OrderStateUse, "begin_time": time.Now()})
	c.JSON(http.StatusOK, ResultSuccess(res > 0))
}

func (o *OrderController) Stop(c *gin.Context) {
	orderID := c.Query("orderId")
	if lock := repository.LockByOrder(orderID); lock != nil {
		//service.SendCommand(lock.DeviceID(), protocol.OpClosePlaceLock, -1)
		service.ClosePlaceLock(lock.DeviceID())
	}
	if gun := repository.ChargingGunByOrder(orderID); gun != nil {
		service.SendCommand(gun.DeviceID(), protocol.OpCloseEle, -1)
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
	gun := repository.ChargingGunByPrivateOrder(orderID)
	if gun == nil {
		return
	}
	service.OpenGauge(gun.DeviceID(), 0)
}

func (o *OrderController) CloseTwiceOrder(c *gin.Context) {
	orderID := c.Query("orderId")
	c.JSON(http.StatusOK, ResultSuccess(closeTwiceOrder(orderID)))
}

func (o *OrderController) AsyncCloseTwiceGauge(c *gin.Context) {
	orderID := c.Query("orderId")
	go closeGauge(orderID)
	c.JSON(http.StatusOK, ResultSuccess(nil))
}

func (o *OrderController) CloseTwiceOrderWithoutGauge(c *gin.Context) {
	orderID := c.Query("orderId")
	c.JSON(http.StatusOK, ResultSuccess(closeTwiceOrderWithoutGauge(orderID)))
}

func (o *OrderController) SetOrderStateIsWaitReturnGun(c *gin.Context) {
	orderID := c.Query("orderId")
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultError(1, "未找到此订单"))
		return
	}
	if order.State != repository.OrderStateUse {
		c.JSON(http.StatusOK, ResultError(1, "此订单不是使用中状态"))
		return
	}
	gun := repository.ChargingGunByPrivateOrder(orderID)
	if gun != nil {
		service.SendCommand(gun.DeviceID(), protocol.OpCloseEle, -1)
		repository.UpdateChargingDegreeForOrder(orderID)
		orderPrivate := repository.OrderPrivateByOrderid(orderID)
		if orderPrivate != nil {
			var fee decimal.Decimal
			if orderPrivate.IsPrivateUser == 1 {
				fee = repository.PrivateFeesByPrivateOrderid(orderID)
			} else {
				fee = repository.PrivateFeesByOrderid(orderID)
			}
			repository.UpdateOrderBasicConsumption(orderID, fee.Round(2))
		}
	}
	res := repository.UpdateOrderState(orderID, repository.OrderStateWaitGun)
	c.JSON(http.StatusOK, ResultSuccess(res > 0))
}

func (o *OrderController) ManualDetectionIsReturnGun(c *gin.Context) {
	orderID := c.Query("orderId")
	time.Sleep(time.Second)
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if !isZeroTime(order.OverTime) {
		res := repository.UpdateOrderFields(orderID, map[string]interface{}{
			"close_time": time.Now(),
			"state":      repository.OrderStateWaitPay,
		})
		c.JSON(http.StatusOK, ResultSuccess(res > 0))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(false))
}

func (o *OrderController) ByOrderIdSendCloseEleAndMessage(c *gin.Context) {
	orderID := c.Query("orderId")
	if gun := repository.ChargingGunByOrder(orderID); gun != nil {
		if !service.SendCommand(gun.DeviceID(), protocol.OpCloseEle, -1) {
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
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultError(1, "订单不存在: "+orderID))
		return
	}
	needProfitSharing := order.ConsumptionType == repository.ConsumeTypeYiXiang
	appID := ""
	if order.ConsumptionType == repository.ConsumeTypeCaiParking {
		appID = yiparlAppID
	} else {
		appID = appid
	}
	c.JSON(http.StatusOK, ResultSuccess(createPayScoreOrder(orderID, needProfitSharing, appID)))
}

func (o *OrderController) QueryPayScore(c *gin.Context) {
	orderID := c.Query("orderId")
	amount := decimal.NewFromFloat(queryFloat(c, "amount"))

	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(wxPayStateCancel))
		return
	}
	state := queryWxOrderState(orderID)

	switch order.ConsumptionType {
	case repository.ConsumeTypeCaiParking:
		switch state {
		case wxPayStateDone, wxPayStateDoing:
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateBooking,
				map[string]interface{}{"state": repository.OrderStateBooked, "decide_amount": amount})
		case wxPayStateRevoke:
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateBooking,
				map[string]interface{}{"state": repository.OrderStateRevoked, "decide_amount": amount})
		}
	case repository.ConsumeTypeYiXiang:
		switch state {
		case wxPayStateDone, wxPayStateDoing:
			if repository.ConfigByOrderid(orderID) == nil {
				repository.CreateConfig(orderID)
			}
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateBooking,
				map[string]interface{}{"state": repository.OrderStateBooked, "decide_amount": amount})
		case wxPayStateRevoke:
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateBooking,
				map[string]interface{}{"state": repository.OrderStateRevoked, "decide_amount": amount})
		}
	case repository.ConsumeTypeNeighbor:
		switch state {
		case wxPayStateDone, wxPayStateDoing:
			gun := repository.ChargingGunByPrivateOrder(orderID)
			if gun == nil || !service.SendCommand(gun.DeviceID(), protocol.OpOpenCharge, -1) {
				c.JSON(http.StatusOK, ResultError(1, "开充电枪失败"))
				return
			}
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateConduct,
				map[string]interface{}{"state": repository.OrderStateUse, "decide_amount": amount, "begin_time": time.Now()})
		case wxPayStateRevoke:
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateBooking,
				map[string]interface{}{"state": repository.OrderStateRevoked, "decide_amount": amount})
		}
	default: // 二轮车充电
		switch state {
		case wxPayStateDone, wxPayStateDoing:
			if !openGauge(orderID) {
				c.JSON(http.StatusOK, ResultError(1, "开闸失败"))
				return
			}
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateConduct,
				map[string]interface{}{"state": repository.OrderStateUse, "decide_amount": amount})
		case wxPayStateRevoke:
			repository.UpdateOrderFieldsByState(orderID, repository.OrderStateConduct,
				map[string]interface{}{"state": repository.OrderStateRevoked, "decide_amount": amount})
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

	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultError(1, "订单不存在: "+orderID))
		return
	}
	if strings.TrimSpace(order.WechatTransactionID) != "" {
		if order.State != repository.OrderStateFinished {
			repository.UpdateOrderState(orderID, repository.OrderStateFinished)
		}
		c.JSON(http.StatusOK, ResultSuccess(true))
		return
	}

	var fees decimal.Decimal
	var amount int64
	if order.ConsumptionType != repository.ConsumeTypeCaiParking {
		fees = order.BasicConsumption
		if fees.IsZero() && order.ConsumptionType == repository.ConsumeTypeYiXiang {
			updateBasicConsumptionForOrder(orderID)
			if refetched, err := repository.FindOrderByOrderid(orderID); err == nil {
				order = refetched
			}
			fees = order.BasicConsumption
		}
		if fees.IsZero() {
			fees = decimal.Zero
		}
		amount = fees.Mul(decimal.NewFromInt(100)).RoundDown(0).IntPart()
		if repository.UpdateOrderState(orderID, repository.OrderStateFinished) == 0 {
			c.JSON(http.StatusOK, ResultSuccess(false))
			return
		}
		service.DelRedisValue("orderState:" + orderID)
	} else {
		amount = order.TotalPrice.Mul(decimal.NewFromInt(100)).RoundDown(0).IntPart()
	}

	if order.ConsumptionType == repository.ConsumeTypeNeighbor && isPrivateUser == "1" {
		completeWxOrder(orderID, 0)
		c.JSON(http.StatusOK, ResultSuccess(true))
		return
	}

	if amount > 0 {
		completeWxOrder(orderID, int(amount))
		paid := decimal.NewFromInt(amount).Div(decimal.NewFromInt(100))
		repository.UpdateOrderBasicConsumption(orderID, paid)
		repository.InvoiceForOrder(*order, paid)
	} else {
		cancelWxOrder(orderID)
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

func (o *OrderController) FinishBalans(c *gin.Context) {
	orderID := c.Query("orderId")
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		c.JSON(http.StatusOK, ResultError(1, "订单不存在: "+orderID))
		return
	}
	var amount decimal.Decimal
	if order.ConsumptionType != repository.ConsumeTypeCaiParking {
		amount = order.BasicConsumption
		if repository.UpdateOrderState(orderID, repository.OrderStateFinished) == 0 {
			c.JSON(http.StatusOK, ResultSuccess(false))
			return
		}
		service.DelRedisValue("orderState:" + orderID)
	} else {
		amount = order.TotalPrice
	}
	if order.ConsumptionType == repository.ConsumeTypeNeighbor && order.DecideAmount.IsZero() {
		amount = decimal.Zero
	}

	if amount.IsPositive() {
		if !repository.ConsumeBalance(order.Openid, amount) {
			c.JSON(http.StatusOK, ResultError(1, "余额支付失败"))
			return
		}
		repository.UpdateOrderBasicConsumption(orderID, amount)
		repository.InvoiceForOrder(*order, amount)
		if order.ConsumptionType == repository.ConsumeTypeCaiParking {
			repository.SaveBalanceRecord(order.Openid, "停车消费", amount)
		} else {
			repository.SaveBalanceRecord(order.Openid, "充电消费", amount)
		}
	} else {
		repository.CompleteOrderIsamountNot(order.Openid)
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
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil || strings.TrimSpace(order.WechatTransactionID) == "" {
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
	power := decimal.RequireFromString(repository.ChargeGunPowerSmall)
	if gun := repository.ChargingGunByPidAndDirection(productID, direction); gun != nil && gun.Gun.Model == repository.ChargeGunModel220 {
		power = decimal.RequireFromString(repository.ChargeGunPowerSmall)
	} else if gun != nil {
		power = decimal.RequireFromString(repository.ChargeGunPowerBig)
	}
	bc := decimal.NewFromFloat(batteryCapacity)
	charging := decimal.NewFromFloat(chargingQuantity).Div(decimal.NewFromInt(100))
	remaining := decimal.NewFromFloat(remainingQuantity).Div(decimal.NewFromInt(100))
	coef := decimal.RequireFromString(repository.ChargeCoefficient)
	result := bc.Mul(charging.Sub(remaining)).DivRound(power, 2).Mul(coef).Round(2)
	c.JSON(http.StatusOK, ResultSuccess(result))
}

func (o *OrderController) AddImage(c *gin.Context) {
	// Java：压缩图片 → ImageService.add（COS 存储）→ 关联到手机号对应 openid 的最新订单 image_id。
	// 此处省略压缩与 COS 上传，仅记录。
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ 内部结算工具 ============

// isAfterNextMidnight 判断时间是否在凌晨 0 点以后、8 点以前（对齐 Java DateTimeUtil.isAfterNextMidnight）。
func isAfterNextMidnight(t time.Time) bool {
	return t.Hour() < 8
}

// calculateOvertimeFee 计算驿享充电超时占位费（对齐 Java calculateOvertimeFee）。
func calculateOvertimeFee(order model.OrderTbl, fee decimal.Decimal, orderID string) decimal.Decimal {
	var chargeEndTime time.Time
	if !isZeroTime(order.FullTime) {
		chargeEndTime = timeOrZero(order.FullTime)
	} else if !isZeroTime(order.GunDisconnectTime) {
		chargeEndTime = timeOrZero(order.GunDisconnectTime)
	} else {
		return fee
	}
	var leaveTime time.Time
	if !isZeroTime(order.LeaveTime) {
		leaveTime = timeOrZero(order.LeaveTime)
	} else {
		if repository.LockByOrder(orderID) == nil {
			leaveTime = timeOrZero(order.OverTime)
		} else {
			return fee // 正在检测车辆是否正确驶离
		}
	}
	betweenMinute := leaveTime.Sub(chargeEndTime).Minutes()
	if isAfterNextMidnight(chargeEndTime) {
		eightAM := time.Date(chargeEndTime.Year(), chargeEndTime.Month(), chargeEndTime.Day(), 8, 0, 0, 0, chargeEndTime.Location())
		betweenMinuteAfter := eightAM.Sub(chargeEndTime).Minutes()
		if betweenMinute-betweenMinuteAfter > 0 {
			timeoutFee := decimal.NewFromFloat(betweenMinute - betweenMinuteAfter).Mul(decimal.NewFromFloat(repository.OvertimeParkingFee))
			if timeoutFee.GreaterThan(decimal.NewFromFloat(repository.OvertimeServiceFee)) {
				timeoutFee = decimal.NewFromFloat(repository.OvertimeServiceFee)
			}
			fee = timeoutFee.Add(fee)
		}
	} else if betweenMinute > 30 {
		overtimeMinutes := betweenMinute - 30
		timeoutFee := decimal.NewFromFloat(overtimeMinutes).Mul(decimal.NewFromFloat(repository.OvertimeParkingFee))
		if timeoutFee.GreaterThan(decimal.NewFromFloat(repository.OvertimeServiceFee)) {
			timeoutFee = decimal.NewFromFloat(repository.OvertimeServiceFee)
		}
		fee = timeoutFee.Add(fee)
	}
	return fee
}

// calculateNeighborOvertimeFee 计算邻享充电超时占位费（对齐 Java calculateNeighborOvertimeFee）。
func calculateNeighborOvertimeFee(order *model.OrderTbl) decimal.Decimal {
	orderPrivate := repository.OrderPrivateByOrderid(order.Orderid)
	if orderPrivate == nil || orderPrivate.IsPrivateUser != 2 {
		return decimal.Zero
	}
	privatePlace := repository.PrivatePlaceByID(order.SpacesID)
	if privatePlace == nil {
		return decimal.Zero
	}
	chargingBean := repository.PrivateChargingBySpacesNum(privatePlace.SpacesCode)
	if chargingBean == nil {
		return decimal.Zero
	}
	// 桩主需要换停车位且收取超时占位费
	if chargingBean.ExchangePlace && chargingBean.OvertimeFee {
		if isZeroTime(order.OverTime) || (isZeroTime(order.NeighborRelocateTime) && isZeroTime(order.MoveCarTime)) {
			return decimal.Zero
		}
		var relocateTime time.Time
		if !isZeroTime(order.NeighborRelocateTime) && !isZeroTime(order.MoveCarTime) {
			if timeOrZero(order.NeighborRelocateTime).Before(timeOrZero(order.MoveCarTime)) {
				relocateTime = timeOrZero(order.NeighborRelocateTime)
			} else {
				relocateTime = timeOrZero(order.MoveCarTime)
			}
		} else if !isZeroTime(order.NeighborRelocateTime) {
			relocateTime = timeOrZero(order.NeighborRelocateTime)
		} else {
			relocateTime = timeOrZero(order.MoveCarTime)
		}
		return neighborOvertimeFeeFromTime(relocateTime, timeOrZero(order.OverTime))
	}
	// 桩主不需要换停车位但收取超时占位费：以充满时间为起始
	if chargingBean.OvertimeFee {
		if isZeroTime(order.FullTime) || isZeroTime(order.OverTime) {
			return decimal.Zero
		}
		return neighborOvertimeFeeFromTime(timeOrZero(order.FullTime), timeOrZero(order.OverTime))
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
		timeoutFee := decimal.NewFromFloat(after8 - 30).Mul(decimal.NewFromFloat(repository.OvertimeParkingFee))
		if timeoutFee.GreaterThan(decimal.NewFromFloat(repository.OvertimeServiceFee)) {
			timeoutFee = decimal.NewFromFloat(repository.OvertimeServiceFee)
		}
		return timeoutFee
	}
	if betweenMinute <= 30 {
		return decimal.Zero
	}
	timeoutFee := decimal.NewFromFloat(betweenMinute - 30).Mul(decimal.NewFromFloat(repository.OvertimeParkingFee))
	if timeoutFee.GreaterThan(decimal.NewFromFloat(repository.OvertimeServiceFee)) {
		timeoutFee = decimal.NewFromFloat(repository.OvertimeServiceFee)
	}
	return timeoutFee
}

// updateBasicConsumptionForOrder 计算并更新订单基本消费（对齐 Java updateBasicConsumption）。
func updateBasicConsumptionForOrder(orderID string) {
	fee, _ := electronicFeesByOrderid(orderID)
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		return
	}
	if order.ChargeGunPull == 0 && isZeroTime(order.OverTime) {
		beginTime := timeOrZero(order.BeginTime)
		var nowTime time.Time
		if !isZeroTime(order.LeaveTime) {
			nowTime = timeOrZero(order.LeaveTime)
		} else {
			if repository.LockByOrder(orderID) == nil {
				repository.UpdateOrderBasicConsumption(orderID, fee.Round(2))
				return
			}
			return // 正在检测车辆是否正确驶离
		}
		betweenMinute := nowTime.Sub(beginTime).Minutes()
		overtimeMinutes := betweenMinute - 15
		if overtimeMinutes < 0 {
			overtimeMinutes = 0
		}
		chargeFee := decimal.NewFromFloat(overtimeMinutes).Mul(decimal.NewFromFloat(repository.OvertimeParkingFee))
		if chargeFee.GreaterThan(decimal.NewFromFloat(repository.OvertimeServiceFee)) {
			chargeFee = decimal.NewFromFloat(repository.OvertimeServiceFee)
		}
		fee = chargeFee.Add(fee)
	} else if fee.IsZero() {
		beginTime := timeOrZero(order.BeginTime)
		var nowTime time.Time
		if !isZeroTime(order.LeaveTime) {
			nowTime = timeOrZero(order.LeaveTime)
		} else {
			if repository.LockByOrder(orderID) == nil {
				nowTime = timeOrZero(order.OverTime)
			} else {
				return // 正在检测车辆是否正确驶离
			}
		}
		betweenMinute := nowTime.Sub(beginTime).Minutes()
		overtimeMinutes := betweenMinute - 15
		if overtimeMinutes < 0 {
			overtimeMinutes = 0
		}
		chargeFee := decimal.NewFromFloat(overtimeMinutes).Mul(decimal.NewFromFloat(repository.OvertimeParkingFee))
		if chargeFee.GreaterThan(decimal.NewFromFloat(repository.OvertimeServiceFee)) {
			chargeFee = decimal.NewFromFloat(repository.OvertimeServiceFee)
		}
		fee = chargeFee.Add(fee)
	} else {
		if order.ConsumptionType != repository.ConsumeTypeNeighbor && (!isZeroTime(order.FullTime) || !isZeroTime(order.GunDisconnectTime)) {
			if order.ConsumptionType == repository.ConsumeTypeYiXiang && order.SpacesID != 0 {
				if spaces := repository.SpacesByID(order.SpacesID); spaces != nil && !spaces.OvertimeFee {
					// 车位设置不收取超时占位费
				} else {
					fee = calculateOvertimeFee(*order, fee, orderID)
				}
			} else {
				fee = calculateOvertimeFee(*order, fee, orderID)
			}
		}
	}
	repository.UpdateOrderBasicConsumption(orderID, fee.Round(2))
}

// ============ 二轮车计费 ============
// reOrderPowerNoServiceFees 查询订单电费（不含服务费）。被 wechat_pay.go 复用，故保留薄包装。
func reOrderPowerNoServiceFees(orderID string) decimal.Decimal {
	return repository.ReOrderPowerNoServiceFees(orderID)
}

// ceilHours 充电时长分钟数向上取整为小时数（不足 1 小时按 1 小时），对齐 Java chargingHours 计算。
func ceilHours(minutes int64) int64 {
	h := minutes / 60
	if minutes%60 > 0 || h == 0 {
		h++
	}
	return h
}

// closeGauge 二轮车关闸（对齐 Java CloseGAUGE）。
func closeGauge(orderID string) {
	receptacle := repository.ReceptacleByOrderid(orderID)
	if receptacle == nil {
		return
	}
	product := repository.ProductByPid(receptacle.Pid)
	if product == nil {
		return
	}
	rid, _ := strconv.Atoi(strings.Split(receptacle.Name, "号")[0])
	service.CloseGauge(product.Imei, rid)
}

// openGauge 二轮车开闸（对齐 Java OpenGAUGE，含开始时间/状态/插座状态更新）。
func openGauge(orderID string) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		return false
	}
	receptacle := repository.ReceptacleByOrderid(orderID)
	if receptacle == nil {
		return false
	}
	product := repository.ProductByPid(order.Pid)
	if product == nil {
		return false
	}
	rid, _ := strconv.Atoi(strings.Split(receptacle.Name, "号")[0])
	if !service.OpenGauge(product.Imei, rid) {
		return false
	}
	// 始终创建订单电量记录（对齐 Java reOrderPowerService.add 的幂等插入）。
	if repository.CountReOrderElectronicsByOrderID(orderID) == 0 {
		startValue := decimal.Zero
		if rp := repository.ReceptaclePowerByPidAndRid(product.Pid, strconv.Itoa(rid)); rp != nil {
			startValue = rp.MeterValue
		}
		repository.InsertReOrderElectronic(orderID, startValue)
	}
	repository.UpdateOrderFields(orderID, map[string]interface{}{
		"begin_time": time.Now(),
		"state":      repository.OrderStateUse,
	})
	repository.ActivateReceptacle(receptacle.ID)
	return true
}

// closeTwiceOrder 结束二轮车充电订单（带关闸），对齐 Java closeTwiceOrder。
func closeTwiceOrder(orderID string) bool {
	return settleTwiceOrder(orderID, true)
}

// closeTwiceOrderWithoutGauge 结束二轮车充电订单（不关闸，关闸由 asyncCloseTwiceGauge 异步执行），
// 对齐 Java closeTwiceOrderWithoutGauge（套餐时长不足时对超出部分按比例计费）。
func closeTwiceOrderWithoutGauge(orderID string) bool {
	return settleTwiceOrder(orderID, false)
}

// settleTwiceOrder 二轮车订单结算：计算电费/服务费 → 关闸 → 扣款 → 落库（整体事务）。
func settleTwiceOrder(orderID string, withGauge bool) bool {
	order, err := repository.FindOrderByOrderid(orderID)
	if err != nil {
		return false
	}

	meterValue := repository.ReOrderPowerValue(orderID).Round(2)
	now := time.Now()

	// 套餐抵扣判断
	hasValidPackage := false
	var chargingMinutes int64
	var packageRecord *model.PackageRecordTbl
	if pr, err := repository.LatestValidPackageByOpenid(order.Openid); err == nil {
		packageRecord = pr
		orderMaxPower := repository.ReOrderPowerMaxPower(orderID)
		packageMaxPower := float64(pr.MaxPower)
		if packageMaxPower > 0 && orderMaxPower > packageMaxPower {
			hasValidPackage = false
		} else if pr.Duration == -1 {
			hasValidPackage = true
		} else if pr.Duration > 0 {
			hasValidPackage = true
			if !isZeroTime(order.BeginTime) {
				chargingMinutes = int64(time.Since(timeOrZero(order.BeginTime)).Minutes())
				if chargingMinutes < 1 {
					chargingMinutes = 1
				}
			}
		}
	}

	var fee, calculatedServiceFee, powerRate decimal.Decimal
	var newPackageDuration *int32 // 非 nil 表示需在事务内更新套餐剩余时长
	if hasValidPackage {
		switch {
		case packageRecord.Duration == -1:
			// 无限时长套餐：免费，不扣时长
			fee, calculatedServiceFee, powerRate = decimal.Zero, decimal.Zero, decimal.Zero
		case packageRecord.Duration > 0:
			remaining := int64(packageRecord.Duration) - chargingMinutes
			if remaining < 0 {
				// 超出套餐时长
				if !withGauge {
					// closeTwiceOrderWithoutGauge：超出部分按比例计费
					overtimeMinutes := -remaining
					totalPowerFee := repository.ReOrderPowerNoServiceFees(orderID)
					var overtimePowerFee decimal.Decimal
					if chargingMinutes > 0 && totalPowerFee.IsPositive() {
						overtimePowerFee = totalPowerFee.Mul(decimal.NewFromInt(overtimeMinutes)).
							Div(decimal.NewFromInt(chargingMinutes)).Round(2)
					}
					calculatedServiceFee = repository.ServiceFeeByPowerRange(repository.ReOrderPowerMaxPower(orderID)).
						Mul(decimal.NewFromInt(ceilHours(overtimeMinutes)))
					if repository.ReOrderPowerValue(orderID).IsZero() && totalPowerFee.IsZero() {
						calculatedServiceFee = decimal.Zero
						overtimePowerFee = decimal.Zero
					}
					powerRate = overtimePowerFee
					fee = overtimePowerFee.Add(calculatedServiceFee)
				} else {
					// closeTwiceOrder（带关闸）：超出部分免费，时长钳制为 0
					fee, calculatedServiceFee, powerRate = decimal.Zero, decimal.Zero, decimal.Zero
				}
				zero := int32(0)
				newPackageDuration = &zero
			} else {
				// 套餐时长足够：免费，扣除时长
				fee, calculatedServiceFee, powerRate = decimal.Zero, decimal.Zero, decimal.Zero
				rem := int32(remaining)
				newPackageDuration = &rem
			}
		}
	} else if repository.UserIsFreeServiceByOrder(orderID) {
		fee = repository.ReOrderPowerNoServiceFees(orderID)
		calculatedServiceFee = decimal.Zero
		powerRate = fee
	} else {
		betweenMinute := int64(0)
		if !isZeroTime(order.BeginTime) {
			betweenMinute = int64(time.Since(timeOrZero(order.BeginTime)).Minutes())
		}
		calculatedServiceFee = repository.ServiceFeeByPowerRange(repository.ReOrderPowerMaxPower(orderID)).
			Mul(decimal.NewFromInt(ceilHours(betweenMinute)))
		logger.Mylog.Info().Msgf("calculatedServiceFee ========>>> %v", calculatedServiceFee)
		fee = repository.ReOrderPowerNoServiceFees(orderID)
		logger.Mylog.Info().Msgf("Fee ========>>> %v", fee)
		powerRate = fee
		if repository.ReOrderPowerValue(orderID).IsZero() && powerRate.IsZero() {
			calculatedServiceFee = decimal.Zero
		}
		fee = fee.Add(calculatedServiceFee)
	}

	// 关闸（带关闸的路由）
	if withGauge {
		closeGauge(orderID)
	}

	// 插座状态与充电时长（落库前备好）
	var receptacleID int32
	if receptacle := repository.ReceptacleByOrderid(orderID); receptacle != nil {
		receptacleID = receptacle.ID
	}
	var chargingTime string
	if !isZeroTime(order.BeginTime) {
		chargingTime = time.Time{}.Add(time.Since(timeOrZero(order.BeginTime))).Format("15:04:05")
	}

	packageOrderNo := ""
	if newPackageDuration != nil {
		packageOrderNo = packageRecord.OrderNo
	}

	// 落库 + 扣款（整体事务，对齐 Java transactionManager，任何一步失败整体回滚）
	state := queryWxOrderState(orderID)
	err = repository.SettleTwiceOrderTx(
		orderID, order.Openid,
		powerRate.Round(2), calculatedServiceFee.Round(2), fee, meterValue,
		newPackageDuration, packageOrderNo, receptacleID, chargingTime, now,
		state != wxPayStateCancel, completeWxOrder,
	)
	return err == nil
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
