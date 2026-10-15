package repository

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

func sumBasicConsumption(db *gorm.DB) decimal.Decimal {
	var v decimal.Decimal
	if err := db.Select("COALESCE(SUM(basic_consumption), 0)").Row().Scan(&v); err != nil {
		return decimal.Zero
	}
	return v
}

// RevenueSpaces 根据 type 解析消费类型与车位 id。
func RevenueSpaces(spacesCode, typ string) (int32, string, bool) {
	switch typ {
	case "驿享充电":
		var s model.TabParkingSpace
		if err := conf.Db.Where("spaces_code = ?", spacesCode).First(&s).Error; err != nil {
			return 0, "", false
		}
		return s.ID, "驿享充电", true
	case "邻享充电":
		var s model.PrivatePlaceTbl
		if err := conf.Db.Where("spaces_code = ?", spacesCode).First(&s).Error; err != nil {
			return 0, "", false
		}
		return s.ID, "邻享充电", true
	default:
		return 0, "", false
	}
}

// RevenueOrders 查询对应已完成订单（begin_time 降序）。
func RevenueOrders(spacesCode, typ string) ([]model.OrderTbl, string, bool) {
	spacesID, consumptionType, ok := RevenueSpaces(spacesCode, typ)
	if !ok {
		return nil, "", false
	}
	var orders []model.OrderTbl
	conf.Db.Model(&model.OrderTbl{}).
		Where("spaces_id = ? AND consumption_type = ? AND state = ?", spacesID, consumptionType, "已完成").
		Order("begin_time desc").Find(&orders)
	return orders, consumptionType, true
}

// CountPlacesNotState 统计状态不为指定值的车位。
func CountPlacesNotState(state string) int64 {
	var max int64
	conf.Db.Model(&model.PlaceTbl{}).Where("state != ?", state).Count(&max)
	return max
}

// PagePlacesNotState 分页查询状态不为指定值的车位。
func PagePlacesNotState(state string, offset, limit int) []model.PlaceTbl {
	var places []model.PlaceTbl
	conf.Db.Model(&model.PlaceTbl{}).Where("state != ?", state).Offset(offset).Limit(limit).Find(&places)
	return places
}

// SumFinishedOrdersBasicConsumption 求和已完成订单 basic_consumption。
func SumFinishedOrdersBasicConsumption() decimal.Decimal {
	return sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).Where("state = ?", "已完成"))
}

// SumFinishedOrdersTodayBasicConsumption 求和今日已完成订单 basic_consumption。
func SumFinishedOrdersTodayBasicConsumption() decimal.Decimal {
	today := time.Now().Format("2006-01-02")
	return sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("state = ?", "已完成").Where("over_time LIKE ?", today+"%"))
}

// FinishedOrdersByLockid 查询已完成订单（按锁），返回总数与列表。
func FinishedOrdersByLockid(lockid string, offset, limit int) (int64, []model.OrderTbl) {
	var max int64
	conf.Db.Model(&model.OrderTbl{}).Where("state = ? AND lockid = ?", "已完成", lockid).Count(&max)
	var orders []model.OrderTbl
	conf.Db.Model(&model.OrderTbl{}).Where("state = ? AND lockid = ?", "已完成", lockid).
		Offset(offset).Limit(limit).Find(&orders)
	return max, orders
}

// SumFinishedOrdersByLockidBasicConsumption 按锁求和已完成订单 basic_consumption。
func SumFinishedOrdersByLockidBasicConsumption(lockid string) decimal.Decimal {
	return sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).Where("state = ? AND lockid = ?", "已完成", lockid))
}

// SumFinishedOrdersByLockidTodayBasicConsumption 按锁求和今日已完成订单 basic_consumption。
func SumFinishedOrdersByLockidTodayBasicConsumption(lockid string) decimal.Decimal {
	today := time.Now().Format("2006-01-02")
	return sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("state = ? AND lockid = ?", "已完成", lockid).Where("over_time LIKE ?", today+"%"))
}

// DeleteOrderByOrderid 删除订单。
func DeleteOrderByOrderid(orderid string) {
	conf.Db.Where("orderid = ?", orderid).Delete(&model.OrderTbl{})
}

// SumFinishedOrdersBySpacesBasicConsumption 按车位与消费类型求和已完成订单 basic_consumption。
func SumFinishedOrdersBySpacesBasicConsumption(spacesID int32, consumptionType string) decimal.Decimal {
	return sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("spaces_id = ? AND consumption_type = ? AND state = ?", spacesID, consumptionType, "已完成"))
}

// SumFinishedOrdersBySpacesTodayBasicConsumption 按车位与消费类型求和今日已完成订单 basic_consumption。
func SumFinishedOrdersBySpacesTodayBasicConsumption(spacesID int32, consumptionType string) decimal.Decimal {
	today := time.Now().Format("2006-01-02")
	return sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("spaces_id = ? AND consumption_type = ? AND state = ?", spacesID, consumptionType, "已完成").
		Where("over_time LIKE ?", "%"+today+"%"))
}

// TodayFinishedOrders 查询今日已完成订单，返回总数、列表与今日总收入。
func TodayFinishedOrders(page, length int) (int64, []model.OrderTbl, decimal.Decimal) {
	today := time.Now().Format("2006-01-02")
	todayQuery := func() *gorm.DB {
		return conf.Db.Model(&model.OrderTbl{}).
			Where("over_time LIKE ? AND state = ?", "%"+today+"%", "已完成").
			Order("over_time desc")
	}
	var max int64
	todayQuery().Count(&max)
	var orders []model.OrderTbl
	db := todayQuery()
	if page > 0 && length > 0 {
		db = db.Offset((page - 1) * length).Limit(length)
	}
	db.Find(&orders)
	todayTotalIncome := sumBasicConsumption(conf.Db.Model(&model.OrderTbl{}).
		Where("over_time LIKE ? AND state = ?", "%"+today+"%", "已完成"))
	return max, orders, todayTotalIncome
}
