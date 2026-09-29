package handler

import (
	"time"

	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
)

// ============================================================================
// 订单相关的共享查询 / 金额 / 余额工具，对齐 Java OrderServiceImpl 依赖的各 Service。
// 均直接复用 conf.Db（GORM），保持与现有 handler 一致的风格。
// ============================================================================

// 订单状态 / 消费类型常量（对齐 Java OrderServiceImpl 中的字段与状态字面量）。
const (
	orderStateBooked   = "已预约" // state 字段
	orderStateBooking  = "预约中" // bookingState 字段
	orderStateConduct  = "进行中"
	orderStateUse      = "使用中"
	orderStateWaitPay  = "待支付"
	orderStateFinished = "已完成"
	orderStateRevoked  = "已取消"
	orderStateWaitUse  = "待使用"
	orderStateWaitGun  = "待还枪"

	consumeTypeCaiParking = "踩踩停车"
	consumeTypeYiXiang    = "驿享充电"
	consumeTypeTwice      = "二轮车充电"
	consumeTypeNeighbor   = "邻享充电"
)

// 业务常量（对齐 Java BusinessConstants）。
const (
	overtimeServiceFee   = 200.0 // 超时占位费封顶（元）
	overtimeParkingFee   = 0.05  // 超时占位费单价（元/分钟）
	chargeGunModel220    = int32(220)
	chargeGunPowerSmall  = "7"    // 220V 充电功率
	chargeGunPowerBig    = "21"   // 380V 充电功率
	chargeCoefficient    = "1.16" // 预计充电时长系数
	orderTwiceChargeRate = "1.0064"
)

// ============================ 设备 / 车位 / 产品 / 网关 ============================

// chargingGunCtx 充电枪及其关联的产品、网关信息。
type chargingGunCtx struct {
	gun     model.TabChargingGun
	product model.TabProduct
	gateway model.TabGateway
}

// deviceID 返回主题用设备标识：优先 4G IMEI，否则 LoRa NID。
func (g *chargingGunCtx) deviceID() string {
	if g.product.Imei != "" {
		return g.product.Imei
	}
	return g.product.Nid
}

func (g *chargingGunCtx) fillProductGateway() {
	if g.gun.ProductID == "" {
		return
	}
	conf.Db.Where("pid = ?", g.gun.ProductID).First(&g.product)
	conf.Db.Where("id = ?", g.product.GatewayID).First(&g.gateway)
}

// chargingGunByOrder 根据订单查询四轮车充电枪（order→tab_parking_spaces→tab_charging_gun）。
// 对齐 Java ChargingGunMapper.getByOrder。
func chargingGunByOrder(orderID string) *chargingGunCtx {
	var gun model.TabChargingGun
	err := conf.Db.Raw(
		`SELECT cg.* FROM order_tbl o
		 LEFT JOIN tab_parking_spaces s ON o.spaces_id = s.id
		 LEFT JOIN tab_charging_gun cg ON s.charging_gun_id = cg.id
		 WHERE o.orderid = ?`, orderID,
	).Scan(&gun).Error
	if err != nil || gun.ID == 0 {
		return nil
	}
	g := &chargingGunCtx{gun: gun}
	g.fillProductGateway()
	return g
}

// chargingGunByPrivateOrder 根据订单查询私桩充电枪（order→private_place_tbl→tab_charging_gun）。
// 对齐 Java ChargingGunMapper.getPrivateChargingByOrder。
func chargingGunByPrivateOrder(orderID string) *chargingGunCtx {
	var gun model.TabChargingGun
	err := conf.Db.Raw(
		`SELECT cg.* FROM order_tbl o
		 LEFT JOIN private_place_tbl s ON o.spaces_id = s.id
		 LEFT JOIN tab_charging_gun cg ON s.charging_gun_id = cg.id
		 WHERE o.orderid = ?`, orderID,
	).Scan(&gun).Error
	if err != nil || gun.ID == 0 {
		return nil
	}
	g := &chargingGunCtx{gun: gun}
	g.fillProductGateway()
	return g
}

// chargingGunByPidAndDirection 按产品与方向查询充电枪（对齐 ChargingGunMapper.getByPidAndDirection）。
func chargingGunByPidAndDirection(pid, direction string) *chargingGunCtx {
	var gun model.TabChargingGun
	err := conf.Db.Where("product_id = ? AND direction = ?", pid, direction).First(&gun).Error
	if err != nil {
		return nil
	}
	g := &chargingGunCtx{gun: gun}
	g.fillProductGateway()
	return g
}

// lockCtx 车位锁及其关联的产品、网关信息。
type lockCtx struct {
	lock    model.LockTbl
	product model.TabProduct
	gateway model.TabGateway
}

func (l *lockCtx) deviceID() string {
	if l.product.Imei != "" {
		return l.product.Imei
	}
	return l.product.Nid
}

// lockByOrder 根据订单查询车位锁（order→tab_parking_spaces→lock_tbl）。
// 对齐 Java LockService.getByOrder。
func lockByOrder(orderID string) *lockCtx {
	var lock model.LockTbl
	err := conf.Db.Raw(
		`SELECT l.* FROM order_tbl o
		 LEFT JOIN tab_parking_spaces ps ON o.spaces_id = ps.id
		 LEFT JOIN lock_tbl l ON ps.lock_id = l.lockid
		 WHERE o.orderid = ?`, orderID,
	).Scan(&lock).Error
	if err != nil || lock.Lockid == "" {
		return nil
	}
	l := &lockCtx{lock: lock}
	if lock.ProductID != "" {
		conf.Db.Where("pid = ?", lock.ProductID).First(&l.product)
		conf.Db.Where("id = ?", l.product.GatewayID).First(&l.gateway)
	}
	return l
}

func productByPid(pid string) *model.TabProduct {
	var p model.TabProduct
	if conf.Db.Where("pid = ?", pid).First(&p).Error != nil {
		return nil
	}
	return &p
}

func gatewayByID(id int32) *model.TabGateway {
	var g model.TabGateway
	if conf.Db.Where("id = ?", id).First(&g).Error != nil {
		return nil
	}
	return &g
}

// ============================ 车位（tab_parking_spaces / place_tbl） ============================

func spacesByID(id int32) *model.TabParkingSpace {
	var s model.TabParkingSpace
	if conf.Db.Where("id = ?", id).First(&s).Error != nil {
		return nil
	}
	return &s
}

func spacesByLockID(lockid string) *model.TabParkingSpace {
	var s model.TabParkingSpace
	if conf.Db.Where("lock_id = ?", lockid).First(&s).Error != nil {
		return nil
	}
	return &s
}

func spacesByChargingGunID(chargingGunID int32) *model.TabParkingSpace {
	var s model.TabParkingSpace
	if conf.Db.Where("charging_gun_id = ?", chargingGunID).First(&s).Error != nil {
		return nil
	}
	return &s
}

// chargingGunByID 按主键查询充电枪。
func chargingGunByID(id int32) *model.TabChargingGun {
	var g model.TabChargingGun
	if conf.Db.Where("id = ?", id).First(&g).Error != nil {
		return nil
	}
	return &g
}

// otherSpacesIDBySpacesID 一桩一锁：返回与当前车位共用一个锁的另一个车位 id。
// 对齐 Java ParkingSpacesServiceImpl.getOtherSpacesIdBySpacesId。
func otherSpacesIDBySpacesID(spacesID int32) int32 {
	spaces := spacesByID(spacesID)
	if spaces == nil || spaces.LockID == "" || spaces.ChargingGunID == 0 {
		return 0
	}
	gun := chargingGunByID(spaces.ChargingGunID)
	if gun == nil {
		return 0
	}
	otherDirection := "R"
	if gun.Direction == "L" {
		otherDirection = "R"
	} else {
		otherDirection = "L"
	}
	otherGun := chargingGunByPidAndDirection(gun.ProductID, otherDirection)
	if otherGun == nil {
		return 0
	}
	otherSpaces := spacesByChargingGunID(otherGun.gun.ID)
	if otherSpaces == nil {
		return 0
	}
	return otherSpaces.ID
}

// isOneLockPerSpot 判断车位是否为「一桩一锁」（对齐 Java ParkingSpacesServiceImpl.isOneLockPerSpot）。
func isOneLockPerSpot(spacesID int32) bool {
	spaces := spacesByID(spacesID)
	if spaces == nil || spaces.LockID == "" || spaces.ChargingGunID == 0 {
		return false
	}
	gun := chargingGunByID(spaces.ChargingGunID)
	if gun == nil {
		return false
	}
	otherDirection := "R"
	if gun.Direction == "L" {
		otherDirection = "R"
	} else {
		otherDirection = "L"
	}
	otherGun := chargingGunByPidAndDirection(gun.ProductID, otherDirection)
	if otherGun == nil {
		return false
	}
	otherSpaces := spacesByChargingGunID(otherGun.gun.ID)
	if otherSpaces == nil || otherSpaces.LockID == "" {
		return false
	}
	return spaces.LockID == otherSpaces.LockID
}

func spacesByPidAndDirection(pid, direction string) *model.TabParkingSpace {
	var s model.TabParkingSpace
	// 对齐 Java ParkingSpacesServiceImpl.getByPidAndDirection（按充电枪 product_id+direction 关联车位）
	if err := conf.Db.Raw(
		`SELECT s.* FROM tab_parking_spaces s
		 LEFT JOIN tab_charging_gun cg ON s.charging_gun_id = cg.id
		 WHERE cg.product_id = ? AND cg.direction = ?`, pid, direction,
	).Scan(&s).Error; err != nil || s.ID == 0 {
		return nil
	}
	return &s
}

func placeByLockid(lockid string) *model.PlaceTbl {
	var p model.PlaceTbl
	if conf.Db.Where("placeid = (SELECT placeid FROM lock_tbl WHERE lockid = ?)", lockid).First(&p).Error != nil {
		return nil
	}
	return &p
}

// ============================ 私桩 ============================

func orderPrivateByOrderid(orderid string) *model.OrderPrivateTbl {
	var op model.OrderPrivateTbl
	if conf.Db.Where("orderid = ?", orderid).First(&op).Error != nil {
		return nil
	}
	return &op
}

func privatePlaceByID(id int32) *model.PrivatePlaceTbl {
	var p model.PrivatePlaceTbl
	if conf.Db.Where("id = ?", id).First(&p).Error != nil {
		return nil
	}
	return &p
}

func privateChargingBySpacesNum(spacesNum string) *model.PrivateChargingTbl {
	var p model.PrivateChargingTbl
	if conf.Db.Where("spaces_num = ?", spacesNum).First(&p).Error != nil {
		return nil
	}
	return &p
}

// ============================ 参数 / 状态 ============================

func configByOrderid(orderid string) *model.ConfigTbl {
	var c model.ConfigTbl
	if conf.Db.Where("orderid = ?", orderid).First(&c).Error != nil {
		return nil
	}
	return &c
}

func chargingStateByProductAndDirection(productID, direction string) *model.ChargingStateTbl {
	var c model.ChargingStateTbl
	if conf.Db.Where("product_id = ? AND direction = ?", productID, direction).First(&c).Error != nil {
		return nil
	}
	return &c
}

// ============================ 二轮车插座 ============================

func receptacleByID(id int32) *model.ReceptacleTbl {
	var r model.ReceptacleTbl
	if conf.Db.Where("id = ?", id).First(&r).Error != nil {
		return nil
	}
	return &r
}

// receptacleByOrderid 根据订单查询二轮车插座（通过订单 lockid 关联插座名）。
func receptacleByOrderid(orderid string) *model.ReceptacleTbl {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderid).First(&order).Error != nil {
		return nil
	}
	if order.Lockid == "" {
		return nil
	}
	var r model.ReceptacleTbl
	if conf.Db.Where("name = ?", order.Lockid+"号").First(&r).Error != nil {
		return nil
	}
	return &r
}

func receptaclePowerByPidAndRid(pid string, rid string) *model.ReceptaclePowerTbl {
	var rp model.ReceptaclePowerTbl
	if conf.Db.Where("pid = ? AND receptacle_id = ?", pid, rid).First(&rp).Error != nil {
		return nil
	}
	return &rp
}

// ============================ 预约次数 ============================

func driverUseByOpenid(openid string) *model.DriveruseTbl {
	var d model.DriveruseTbl
	if conf.Db.Where("openid = ?", openid).First(&d).Error != nil {
		return nil
	}
	return &d
}

// ============================ 电价（对齐 PriceTimeServiceImpl.getNow） ============================

// currentChargingRate 返回当前时段电价 = total + service*0.8。
func currentChargingRate() decimal.Decimal {
	season := currentSeason()
	now := time.Now().Format("15:04:05")
	var total, service decimal.Decimal
	sql := `SELECT p.total, p.service
		FROM tab_price_time_` + season + ` t
		LEFT JOIN tab_price p ON p.id = t.price_id
		WHERE t.start_time < ? AND t.over_time >= ?
		ORDER BY t.over_time DESC LIMIT 1`
	if err := conf.Db.Raw(sql, now, now).Row().Scan(&total, &service); err != nil {
		return decimal.Zero
	}
	return total.Add(service.Mul(decimal.NewFromFloat(0.8)))
}

// ============================ 余额操作（对齐 UserServiceImpl / BalanceConsumptionServiceImpl） ============================

// freezeBalance 冻结余额：可用余额 → 冻结余额。
func freezeBalance(openid string, amount decimal.Decimal) bool {
	var user model.UserTbl
	if conf.Db.Where("openid = ?", openid).First(&user).Error != nil {
		return false
	}
	if user.Balans.LessThan(amount) {
		return false
	}
	newBalans := user.Balans.Sub(amount)
	newFreeze := user.FreezeBalans.Add(amount)
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).
		Updates(map[string]interface{}{"balans": newBalans, "freeze_balans": newFreeze})
	return res.RowsAffected > 0
}

// consumeBalance 余额消费：按充值订单创建时间 FIFO 扣减 remaining_refund_amount，并释放冻结余额。
func consumeBalance(openid string, amount decimal.Decimal) bool {
	var user model.UserTbl
	if conf.Db.Where("openid = ?", openid).First(&user).Error != nil {
		return false
	}
	totalBalance := user.Balans.Add(user.FreezeBalans)
	if totalBalance.LessThan(amount) {
		return false
	}

	var rechargeOrders []model.RechargeOrderTbl
	conf.Db.Where("openid = ? AND order_status = ?", openid, "已支付").
		Order("create_time asc").Find(&rechargeOrders)

	var rechargeTotal decimal.Decimal
	for _, o := range rechargeOrders {
		rechargeTotal = rechargeTotal.Add(o.RemainingRefundAmount)
	}
	otherSource := totalBalance.Sub(rechargeTotal)
	if otherSource.IsNegative() {
		otherSource = decimal.Zero
	}
	needFromRecharge := amount.Sub(otherSource)
	if needFromRecharge.IsNegative() {
		needFromRecharge = decimal.Zero
	}

	if needFromRecharge.IsPositive() {
		remaining := needFromRecharge
		for i := range rechargeOrders {
			if !remaining.IsPositive() {
				break
			}
			o := &rechargeOrders[i]
			if !o.RemainingRefundAmount.IsPositive() {
				continue
			}
			var deduct decimal.Decimal
			if o.RemainingRefundAmount.GreaterThanOrEqual(remaining) {
				deduct = remaining
				o.RemainingRefundAmount = o.RemainingRefundAmount.Sub(deduct)
				remaining = decimal.Zero
			} else {
				deduct = o.RemainingRefundAmount
				o.RemainingRefundAmount = decimal.Zero
				remaining = remaining.Sub(deduct)
			}
			conf.Db.Model(&model.RechargeOrderTbl{}).Where("order_id = ?", o.OrderID).
				Update("remaining_refund_amount", o.RemainingRefundAmount)
		}
	}

	newBalance := totalBalance.Sub(amount)
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).
		Updates(map[string]interface{}{"balans": newBalance, "freeze_balans": decimal.Zero})
	if res.RowsAffected <= 0 {
		return false
	}
	logger.Mylog.Info().Msgf("余额消费完成: openid=%s, 消费金额=%s, 新余额=%s", openid, amount, newBalance)
	return true
}

// completeOrderIsamountNot 完成订单（费用=0）：释放冻结余额。
func completeOrderIsamountNot(openid string) bool {
	var user model.UserTbl
	if conf.Db.Where("openid = ?", openid).First(&user).Error != nil {
		return false
	}
	newBalans := user.Balans.Add(user.FreezeBalans)
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).
		Updates(map[string]interface{}{"balans": newBalans, "freeze_balans": decimal.Zero})
	return res.RowsAffected > 0
}

// completeOrderIsamountNotNull 取消订单扣费（费用>0）：balans = balans - price + freeze_balans。
func completeOrderIsamountNotNull(openid, price string) bool {
	if price == "undefined" || price == "null" {
		price = "0"
	}
	p, err := decimal.NewFromString(price)
	if err != nil {
		p = decimal.Zero
	}
	var user model.UserTbl
	if conf.Db.Where("openid = ?", openid).First(&user).Error != nil {
		return false
	}
	newBalans := user.Balans.Sub(p).Add(user.FreezeBalans).Round(2)
	res := conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).
		Updates(map[string]interface{}{"balans": newBalans, "freeze_balans": decimal.Zero})
	return res.RowsAffected > 0
}

// ============================ 电费计算（复用 order_crud.go 的 electronicFeesByOrderid） ============================

// updateChargingDegree 设置订单充电度数（对齐 Java updateChargingDegree：电量/100）。
func updateChargingDegree(orderid string) {
	var degree decimal.Decimal
	conf.Db.Raw("SELECT COALESCE(SUM(over_value - start_value),0)/100 FROM tab_order_electronic WHERE order_id = ?", orderid).
		Row().Scan(&degree)
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderid).Update("charging_degree", degree)
}

// invoiceForOrder 构造一条与 Java finish/finishBalans 一致的发票记录并落库。
func invoiceForOrder(order model.OrderTbl, amount decimal.Decimal) {
	inv := model.Invoice{
		InvoiceType: "增值税电子普通发票",
		InvoiceDate: order.OverTime,
		FarmTax:     "123456789123456",
		FarmName:    "成都七彩云创信息技术有限公司",
		FarmAddress: "成都市金牛区长东路一号",
		InvoiceCode: "123456789123",
		InvoiceNum:  "123456789123",
		Amount:      amount,
		TaxAmount:   decimal.Zero,
		State:       0,
		Openid:      order.Openid,
	}
	conf.Db.Create(&inv)
}

// saveBalanceRecord 记录余额明细。
func saveBalanceRecord(openid, recordType string, amount decimal.Decimal) {
	conf.Db.Create(&model.TabBalanceRecord{Openid: openid, Type: recordType, Amount: amount})
}

// ============================ 私桩电费（private_charging_price_tbl） ============================
// 对齐 Java OrderElectronicMapper 的 private 变体（LEFT JOIN private_charging_price_tbl，
// 经 tab_price_time_{season}.price_id 关联）。

// privateFeesByOrderid 对齐 getPrivateFeesByOrderid：SUM((over-start)/100 * (price.total + price.service))。
func privateFeesByOrderid(orderid string) decimal.Decimal {
	season := currentSeason()
	var v decimal.Decimal
	sql := `SELECT COALESCE(SUM((over_value - start_value)/100 * (price.total + price.service)),0)
		FROM tab_order_electronic
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = tab_order_electronic.price_time_id
		LEFT JOIN private_charging_price_tbl price ON price.id = t.price_id
		WHERE order_id = ?`
	conf.Db.Raw(sql, orderid).Row().Scan(&v)
	return v.Round(2)
}

// privateFeesByPrivateOrderid 对齐 getPrivateFeesByPrivateOrderid：SUM((over-start)/100 * price.total)。
func privateFeesByPrivateOrderid(orderid string) decimal.Decimal {
	season := currentSeason()
	var v decimal.Decimal
	sql := `SELECT COALESCE(SUM((over_value - start_value)/100 * price.total),0)
		FROM tab_order_electronic
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = tab_order_electronic.price_time_id
		LEFT JOIN private_charging_price_tbl price ON price.id = t.price_id
		WHERE order_id = ?`
	conf.Db.Raw(sql, orderid).Row().Scan(&v)
	return v.Round(2)
}

// privateFeesByOrderidAndPrivateUser 对齐 getPrivateFeesByOrderidAndPrivateUser：
// SUM((over-start)/100 * (price.total + fee))，fee 为私桩服务费率。
func privateFeesByOrderidAndPrivateUser(orderid string, fee decimal.Decimal) decimal.Decimal {
	season := currentSeason()
	var v decimal.Decimal
	sql := `SELECT COALESCE(SUM((over_value - start_value)/100 * (price.total + ?)),0)
		FROM tab_order_electronic
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = tab_order_electronic.price_time_id
		LEFT JOIN private_charging_price_tbl price ON price.id = t.price_id
		WHERE order_id = ?`
	conf.Db.Raw(sql, fee, orderid).Row().Scan(&v)
	return v.Round(2)
}

// privateElectronicByOrderid 对齐 getElectronicByPrivateOrderid：返回 (totalFee, serviceFee)。
func privateElectronicByOrderid(orderid string) (decimal.Decimal, decimal.Decimal) {
	season := currentSeason()
	var totalFee, serviceFee decimal.Decimal
	sql := `SELECT
		COALESCE(SUM((over_value - start_value)/100 * price.total),0),
		COALESCE(SUM((over_value - start_value)/100 * price.service),0)
		FROM tab_order_electronic
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = tab_order_electronic.price_time_id
		LEFT JOIN private_charging_price_tbl price ON price.id = t.price_id
		WHERE order_id = ?`
	conf.Db.Raw(sql, orderid).Row().Scan(&totalFee, &serviceFee)
	return totalFee.Round(2), serviceFee.Round(2)
}

// privateElectronicByOrderidAndPrivateUser 对齐 getElectronicByPrivateOrderidAndPrivateUser：
// totalFee = SUM(... * price.total)，serviceFee = SUM(... * (chargingBean.Fee * 0.8))。
func privateElectronicByOrderidAndPrivateUser(orderid string) (decimal.Decimal, decimal.Decimal) {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderid).First(&order).Error != nil {
		return decimal.Zero, decimal.Zero
	}
	var fee decimal.Decimal
	if privatePlace := privatePlaceByID(order.SpacesID); privatePlace != nil {
		if chargingBean := privateChargingBySpacesNum(privatePlace.SpacesCode); chargingBean != nil {
			fee = chargingBean.Fee
		}
	}
	season := currentSeason()
	var totalFee, serviceFee decimal.Decimal
	sql := `SELECT
		COALESCE(SUM((over_value - start_value)/100 * price.total),0),
		COALESCE(SUM((over_value - start_value)/100 * (? * 0.8)),0)
		FROM tab_order_electronic
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = tab_order_electronic.price_time_id
		LEFT JOIN private_charging_price_tbl price ON price.id = t.price_id
		WHERE order_id = ?`
	conf.Db.Raw(sql, fee, orderid).Row().Scan(&totalFee, &serviceFee)
	return totalFee.Round(2), serviceFee.Round(2)
}
