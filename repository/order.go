// Package repository 数据访问层：封装所有数据库 / 查询操作，供 handler / service 调用。
package repository

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
)

// ============================================================================
// 订单相关的共享查询 / 金额 / 余额工具，对齐 Java OrderServiceImpl 依赖的各 Service。
// ============================================================================

// 订单状态 / 消费类型常量（对齐 Java OrderServiceImpl 中的字段与状态字面量）。
const (
	OrderStateBooked   = "已预约" // state 字段
	OrderStateBooking  = "预约中" // bookingState 字段
	OrderStateConduct  = "进行中"
	OrderStateUse      = "使用中"
	OrderStateWaitPay  = "待支付"
	OrderStateFinished = "已完成"
	OrderStateRevoked  = "已取消"
	OrderStateWaitUse  = "待使用"
	OrderStateWaitGun  = "待还枪"

	ConsumeTypeCaiParking = "踩踩停车"
	ConsumeTypeYiXiang    = "驿享充电"
	ConsumeTypeTwice      = "二轮车充电"
	ConsumeTypeNeighbor   = "邻享充电"
)

// 业务常量（对齐 Java BusinessConstants）。
const (
	OvertimeServiceFee   = 200.0 // 超时占位费封顶（元）
	OvertimeParkingFee   = 0.05  // 超时占位费单价（元/分钟）
	ChargeGunModel220    = int32(220)
	ChargeGunPowerSmall  = "7"    // 220V 充电功率
	ChargeGunPowerBig    = "21"   // 380V 充电功率
	ChargeCoefficient    = "1.16" // 预计充电时长系数
	OrderTwiceChargeRate = "1.0064"
)

// CurrentSeason 对齐 Java SeasonUtil.getSeason()。
func CurrentSeason() string {
	switch int(time.Now().Month()) {
	case 3, 4, 5, 6, 10, 11:
		return "spring"
	case 7, 8, 9:
		return "summer"
	default: // 12,1,2
		return "winter"
	}
}

// ============================ 设备 / 车位 / 产品 / 网关 ============================

// ChargingGunCtx 充电枪及其关联的产品、网关信息。
type ChargingGunCtx struct {
	Gun     model.TabChargingGun
	Product model.TabProduct
	Gateway model.TabGateway
}

// DeviceID 返回主题用设备标识：优先 4G IMEI，否则 LoRa NID。
func (g *ChargingGunCtx) DeviceID() string {
	if g.Product.Imei != "" {
		return g.Product.Imei
	}
	return g.Product.Nid
}

func (g *ChargingGunCtx) fillProductGateway() {
	if g.Gun.ProductID == "" {
		return
	}
	conf.Db.Where("pid = ?", g.Gun.ProductID).First(&g.Product)
	conf.Db.Where("id = ?", g.Product.GatewayID).First(&g.Gateway)
}

// ChargingGunByOrder 根据订单查询四轮车充电枪（order→tab_parking_spaces→tab_charging_gun）。
// 对齐 Java ChargingGunMapper.getByOrder。
func ChargingGunByOrder(orderID string) *ChargingGunCtx {
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
	g := &ChargingGunCtx{Gun: gun}
	g.fillProductGateway()
	return g
}

// ChargingGunByPrivateOrder 根据订单查询私桩充电枪（order→private_place_tbl→tab_charging_gun）。
// 对齐 Java ChargingGunMapper.getPrivateChargingByOrder。
func ChargingGunByPrivateOrder(orderID string) *ChargingGunCtx {
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
	g := &ChargingGunCtx{Gun: gun}
	g.fillProductGateway()
	return g
}

// ChargingGunByPidAndDirection 按产品与方向查询充电枪（对齐 ChargingGunMapper.getByPidAndDirection）。
func ChargingGunByPidAndDirection(pid, direction string) *ChargingGunCtx {
	var gun model.TabChargingGun
	err := conf.Db.Where("product_id = ? AND direction = ?", pid, direction).First(&gun).Error
	if err != nil {
		return nil
	}
	g := &ChargingGunCtx{Gun: gun}
	g.fillProductGateway()
	return g
}

// LockCtx 车位锁及其关联的产品、网关信息。
type LockCtx struct {
	Lock    model.LockTbl
	Product model.TabProduct
	Gateway model.TabGateway
}

// DeviceID 返回主题用设备标识：优先 4G IMEI，否则 LoRa NID。
func (l *LockCtx) DeviceID() string {
	if l.Product.Imei != "" {
		return l.Product.Imei
	}
	return l.Product.Nid
}

// LockByOrder 根据订单查询车位锁（order→tab_parking_spaces→lock_tbl）。
// 对齐 Java LockService.getByOrder。
func LockByOrder(orderID string) *LockCtx {
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
	l := &LockCtx{Lock: lock}
	if lock.ProductID != "" {
		conf.Db.Where("pid = ?", lock.ProductID).First(&l.Product)
		conf.Db.Where("id = ?", l.Product.GatewayID).First(&l.Gateway)
	}
	return l
}

func ProductByPid(pid string) *model.TabProduct {
	var p model.TabProduct
	if conf.Db.Where("pid = ?", pid).First(&p).Error != nil {
		return nil
	}
	return &p
}

func GatewayByID(id int32) *model.TabGateway {
	var g model.TabGateway
	if conf.Db.Where("id = ?", id).First(&g).Error != nil {
		return nil
	}
	return &g
}

// ============================ 车位（tab_parking_spaces / place_tbl） ============================

func SpacesByID(id int32) *model.TabParkingSpace {
	var s model.TabParkingSpace
	if conf.Db.Where("id = ?", id).First(&s).Error != nil {
		return nil
	}
	return &s
}

func SpacesByLockID(lockid string) *model.TabParkingSpace {
	var s model.TabParkingSpace
	if conf.Db.Where("lock_id = ?", lockid).First(&s).Error != nil {
		return nil
	}
	return &s
}

func SpacesByChargingGunID(chargingGunID int32) *model.TabParkingSpace {
	var s model.TabParkingSpace
	if conf.Db.Where("charging_gun_id = ?", chargingGunID).First(&s).Error != nil {
		return nil
	}
	return &s
}

// ChargingGunByID 按主键查询充电枪。
func ChargingGunByID(id int32) *model.TabChargingGun {
	var g model.TabChargingGun
	if conf.Db.Where("id = ?", id).First(&g).Error != nil {
		return nil
	}
	return &g
}

// OtherSpacesIDBySpacesID 一桩一锁：返回与当前车位共用一个锁的另一个车位 id。
// 对齐 Java ParkingSpacesServiceImpl.getOtherSpacesIdBySpacesId。
func OtherSpacesIDBySpacesID(spacesID int32) int32 {
	spaces := SpacesByID(spacesID)
	if spaces == nil || spaces.LockID == "" || spaces.ChargingGunID == 0 {
		return 0
	}
	gun := ChargingGunByID(spaces.ChargingGunID)
	if gun == nil {
		return 0
	}
	otherDirection := "R"
	if gun.Direction == "L" {
		otherDirection = "R"
	} else {
		otherDirection = "L"
	}
	otherGun := ChargingGunByPidAndDirection(gun.ProductID, otherDirection)
	if otherGun == nil {
		return 0
	}
	otherSpaces := SpacesByChargingGunID(otherGun.Gun.ID)
	if otherSpaces == nil {
		return 0
	}
	return otherSpaces.ID
}

// IsOneLockPerSpot 判断车位是否为「一桩一锁」（对齐 Java ParkingSpacesServiceImpl.isOneLockPerSpot）。
func IsOneLockPerSpot(spacesID int32) bool {
	spaces := SpacesByID(spacesID)
	if spaces == nil || spaces.LockID == "" || spaces.ChargingGunID == 0 {
		return false
	}
	gun := ChargingGunByID(spaces.ChargingGunID)
	if gun == nil {
		return false
	}
	otherDirection := "R"
	if gun.Direction == "L" {
		otherDirection = "R"
	} else {
		otherDirection = "L"
	}
	otherGun := ChargingGunByPidAndDirection(gun.ProductID, otherDirection)
	if otherGun == nil {
		return false
	}
	otherSpaces := SpacesByChargingGunID(otherGun.Gun.ID)
	if otherSpaces == nil || otherSpaces.LockID == "" {
		return false
	}
	return spaces.LockID == otherSpaces.LockID
}

func SpacesByPidAndDirection(pid, direction string) *model.TabParkingSpace {
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

func PlaceByLockid(lockid string) *model.PlaceTbl {
	var p model.PlaceTbl
	if conf.Db.Where("placeid = (SELECT placeid FROM lock_tbl WHERE lockid = ?)", lockid).First(&p).Error != nil {
		return nil
	}
	return &p
}

// ============================ 私桩 ============================

func OrderPrivateByOrderid(orderid string) *model.OrderPrivateTbl {
	var op model.OrderPrivateTbl
	if conf.Db.Where("orderid = ?", orderid).First(&op).Error != nil {
		return nil
	}
	return &op
}

func PrivatePlaceByID(id int32) *model.PrivatePlaceTbl {
	var p model.PrivatePlaceTbl
	if conf.Db.Where("id = ?", id).First(&p).Error != nil {
		return nil
	}
	return &p
}

func PrivateChargingBySpacesNum(spacesNum string) *model.PrivateChargingTbl {
	var p model.PrivateChargingTbl
	if conf.Db.Where("spaces_num = ?", spacesNum).First(&p).Error != nil {
		return nil
	}
	return &p
}

// ============================ 参数 / 状态 ============================

func ConfigByOrderid(orderid string) *model.ConfigTbl {
	var c model.ConfigTbl
	if conf.Db.Where("orderid = ?", orderid).First(&c).Error != nil {
		return nil
	}
	return &c
}

func ChargingStateByProductAndDirection(productID, direction string) *model.ChargingStateTbl {
	var c model.ChargingStateTbl
	if conf.Db.Where("product_id = ? AND direction = ?", productID, direction).First(&c).Error != nil {
		return nil
	}
	return &c
}

// ============================ 二轮车插座 ============================

func ReceptacleByID(id int32) *model.ReceptacleTbl {
	var r model.ReceptacleTbl
	if conf.Db.Where("id = ?", id).First(&r).Error != nil {
		return nil
	}
	return &r
}

// ReceptacleByOrderid 根据订单查询二轮车插座（对齐 Java selectReceptacleByOrderid：number = order.lockid AND pid = order.pid）。
func ReceptacleByOrderid(orderid string) *model.ReceptacleTbl {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderid).First(&order).Error != nil {
		return nil
	}
	if order.Lockid == "" {
		return nil
	}
	var r model.ReceptacleTbl
	if conf.Db.Where("number = ? AND pid = ?", order.Lockid, order.Pid).First(&r).Error != nil {
		return nil
	}
	return &r
}

func ReceptaclePowerByPidAndRid(pid string, rid string) *model.ReceptaclePowerTbl {
	var rp model.ReceptaclePowerTbl
	if conf.Db.Where("pid = ? AND receptacle_id = ?", pid, rid).First(&rp).Error != nil {
		return nil
	}
	return &rp
}

// ============================ 预约次数 ============================

func DriverUseByOpenid(openid string) *model.DriveruseTbl {
	var d model.DriveruseTbl
	if conf.Db.Where("openid = ?", openid).First(&d).Error != nil {
		return nil
	}
	return &d
}

// ============================ 电价（对齐 PriceTimeServiceImpl.getNow） ============================

// CurrentChargingRate 返回当前时段电价 = total + service*0.8。
func CurrentChargingRate() decimal.Decimal {
	season := CurrentSeason()
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

// FreezeBalance 冻结余额：可用余额 → 冻结余额。
// 单条原子 UPDATE 完成「校验 balans>=amount → balans-amount → freeze_balans+amount」，
// 条件写进 WHERE，避免读-改-写竞态与余额被扣成负数。
func FreezeBalance(openid string, amount decimal.Decimal) bool {
	amt := amount.StringFixed(2)
	res := conf.Db.Exec(
		"UPDATE user_tbl SET balans = balans - CAST(? AS DECIMAL(10,2)), freeze_balans = freeze_balans + CAST(? AS DECIMAL(10,2)) WHERE openid = ? AND balans >= CAST(? AS DECIMAL(10,2))",
		amt, amt, openid, amt,
	)
	if res.Error != nil {
		logger.Mylog.Err(res.Error).Msg("冻结失败")
		return false
	}
	if amount.Equal(decimal.Zero) {
		return true
	}
	return res.RowsAffected > 0
}

// ConsumeBalanceTx 在给定事务内执行余额消费：按充值订单创建时间 FIFO 扣减 remaining_refund_amount，
// 并释放冻结余额。返回是否成功、扣减后的新余额及错误。用户不存在/余额不足返回 ok=false 且 err=nil
// （无写入、无需回滚）；DB 错误返回 err（由外层事务回滚）。供需要把余额扣款与订单结算放进同一事务的调用方使用。
func ConsumeBalanceTx(tx *gorm.DB, openid string, amount decimal.Decimal) (bool, decimal.Decimal, error) {
	var user model.UserTbl
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("openid = ?", openid).First(&user).Error; err != nil {
		return false, decimal.Zero, nil // 用户不存在：无操作
	}
	totalBalance := user.Balans.Add(user.FreezeBalans)
	if totalBalance.LessThan(amount) {
		return false, decimal.Zero, nil // 余额不足：无操作
	}

	var rechargeOrders []model.RechargeOrderTbl
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("openid = ? AND order_status = ?", openid, "已支付").
		Order("create_time asc").Find(&rechargeOrders).Error; err != nil {
		return false, decimal.Zero, err
	}

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
			if err := tx.Model(&model.RechargeOrderTbl{}).Where("order_id = ?", o.OrderID).
				Update("remaining_refund_amount", o.RemainingRefundAmount).Error; err != nil {
				return false, decimal.Zero, err
			}
		}
	}

	newBalance := totalBalance.Sub(amount)
	if err := tx.Model(&model.UserTbl{}).Where("openid = ?", openid).
		Updates(map[string]interface{}{"balans": newBalance, "freeze_balans": decimal.Zero}).Error; err != nil {
		return false, decimal.Zero, err
	}
	return true, newBalance, nil
}

// ConsumeBalance 余额消费：包在独立事务内执行 ConsumeBalanceTx，保证原子性与并发安全。
func ConsumeBalance(openid string, amount decimal.Decimal) bool {
	consumed := false
	var newBalance decimal.Decimal
	_ = conf.Db.Transaction(func(tx *gorm.DB) error {
		ok, nb, err := ConsumeBalanceTx(tx, openid, amount)
		if err != nil {
			return err
		}
		if ok {
			consumed = true
			newBalance = nb
		}
		return nil
	})
	if !consumed {
		return false
	}
	logger.Mylog.Info().Msgf("余额消费完成: openid=%s, 消费金额=%s, 新余额=%s", openid, amount, newBalance)
	return true
}

// CompleteOrderIsamountNot 完成订单（费用=0）：释放冻结余额。
// 单条原子 UPDATE：balans = balans + freeze_balans, freeze_balans = 0。
func CompleteOrderIsamountNot(openid string) bool {
	res := conf.Db.Exec(
		"UPDATE user_tbl SET balans = balans + freeze_balans, freeze_balans = 0 WHERE openid = ?",
		openid,
	)
	return res.RowsAffected > 0
}

// CompleteOrderIsamountNotNull 取消订单扣费（费用>0）：balans = balans - price + freeze_balans，freeze_balans=0。
// 单条原子 UPDATE，金额 CAST 为 DECIMAL 保证精确，避免读-改-写竞态。
func CompleteOrderIsamountNotNull(openid, price string) bool {
	if price == "undefined" || price == "null" {
		price = "0"
	}
	p, err := decimal.NewFromString(price)
	if err != nil {
		p = decimal.Zero
	}
	amt := p.StringFixed(2)
	res := conf.Db.Exec(
		"UPDATE user_tbl SET balans = balans - CAST(? AS DECIMAL(10,2)) + freeze_balans, freeze_balans = 0 WHERE openid = ?",
		amt, openid,
	)
	return res.RowsAffected > 0
}

// ============================ 电费计算 ============================

// UpdateChargingDegree 设置订单充电度数（对齐 Java updateChargingDegree：电量/100）。
func UpdateChargingDegree(orderid string) {
	var degree decimal.Decimal
	conf.Db.Raw("SELECT COALESCE(SUM(over_value - start_value),0)/100 FROM tab_order_electronic WHERE order_id = ?", orderid).
		Row().Scan(&degree)
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", orderid).Update("charging_degree", degree)
}

// InvoiceForOrder 构造一条与 Java finish/finishBalans 一致的发票记录并落库。
func InvoiceForOrder(order model.OrderTbl, amount decimal.Decimal) {
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

// SaveBalanceRecord 记录余额明细。
func SaveBalanceRecord(openid, recordType string, amount decimal.Decimal) {
	conf.Db.Create(&model.TabBalanceRecord{Openid: openid, Type: recordType, Amount: amount})
}

// ============================ 私桩电费（private_charging_price_tbl） ============================
// 对齐 Java OrderElectronicMapper 的 private 变体（LEFT JOIN private_charging_price_tbl，
// 经 tab_price_time_{season}.price_id 关联）。

// PrivateFeesByOrderid 对齐 getPrivateFeesByOrderid：SUM((over-start)/100 * (price.total + price.service))。
func PrivateFeesByOrderid(orderid string) decimal.Decimal {
	season := CurrentSeason()
	var v decimal.Decimal
	sql := `SELECT COALESCE(SUM((over_value - start_value)/100 * (price.total + price.service)),0)
		FROM tab_order_electronic
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = tab_order_electronic.price_time_id
		LEFT JOIN private_charging_price_tbl price ON price.id = t.price_id
		WHERE order_id = ?`
	conf.Db.Raw(sql, orderid).Row().Scan(&v)
	return v.Round(2)
}

// PrivateFeesByPrivateOrderid 对齐 getPrivateFeesByPrivateOrderid：SUM((over-start)/100 * price.total)。
func PrivateFeesByPrivateOrderid(orderid string) decimal.Decimal {
	season := CurrentSeason()
	var v decimal.Decimal
	sql := `SELECT COALESCE(SUM((over_value - start_value)/100 * price.total),0)
		FROM tab_order_electronic
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = tab_order_electronic.price_time_id
		LEFT JOIN private_charging_price_tbl price ON price.id = t.price_id
		WHERE order_id = ?`
	conf.Db.Raw(sql, orderid).Row().Scan(&v)
	return v.Round(2)
}

// PrivateFeesByOrderidAndPrivateUser 对齐 getPrivateFeesByOrderidAndPrivateUser：
// SUM((over-start)/100 * (price.total + fee))，fee 为私桩服务费率。
func PrivateFeesByOrderidAndPrivateUser(orderid string, fee decimal.Decimal) decimal.Decimal {
	season := CurrentSeason()
	var v decimal.Decimal
	sql := `SELECT COALESCE(SUM((over_value - start_value)/100 * (price.total + ?)),0)
		FROM tab_order_electronic
		LEFT JOIN tab_price_time_` + season + ` t ON t.id = tab_order_electronic.price_time_id
		LEFT JOIN private_charging_price_tbl price ON price.id = t.price_id
		WHERE order_id = ?`
	conf.Db.Raw(sql, fee, orderid).Row().Scan(&v)
	return v.Round(2)
}

// PrivateElectronicByOrderid 对齐 getElectronicByPrivateOrderid：返回 (totalFee, serviceFee)。
func PrivateElectronicByOrderid(orderid string) (decimal.Decimal, decimal.Decimal) {
	season := CurrentSeason()
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

// PrivateElectronicByOrderidAndPrivateUser 对齐 getElectronicByPrivateOrderidAndPrivateUser：
// totalFee = SUM(... * price.total)，serviceFee = SUM(... * (chargingBean.Fee * 0.8))。
func PrivateElectronicByOrderidAndPrivateUser(orderid string) (decimal.Decimal, decimal.Decimal) {
	var order model.OrderTbl
	if conf.Db.Where("orderid = ?", orderid).First(&order).Error != nil {
		return decimal.Zero, decimal.Zero
	}
	var fee decimal.Decimal
	if privatePlace := PrivatePlaceByID(order.SpacesID); privatePlace != nil {
		if chargingBean := PrivateChargingBySpacesNum(privatePlace.SpacesCode); chargingBean != nil {
			fee = chargingBean.Fee
		}
	}
	season := CurrentSeason()
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
