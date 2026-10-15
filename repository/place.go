package repository

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// PagePlacesByCity 按城市模糊分页查询车位。
func PagePlacesByCity(city string, offset, limit int) (int64, []model.PlaceTbl) {
	var max int64
	conf.Db.Model(&model.PlaceTbl{}).Where("city LIKE ?", "%"+city+"%").Count(&max)
	var places []model.PlaceTbl
	conf.Db.Model(&model.PlaceTbl{}).Where("city LIKE ?", "%"+city+"%").Offset(offset).Limit(limit).Find(&places)
	return max, places
}

// PagePlacesByCityState 按城市与状态分页查询车位。
func PagePlacesByCityState(city, state string, offset, limit int) (int64, []model.PlaceTbl) {
	var max int64
	conf.Db.Model(&model.PlaceTbl{}).Where("city = ? AND state = ?", city, state).Count(&max)
	var places []model.PlaceTbl
	conf.Db.Model(&model.PlaceTbl{}).Where("city = ? AND state = ?", city, state).Offset(offset).Limit(limit).Find(&places)
	return max, places
}

// PagePlacesByState 按状态分页查询车位。
func PagePlacesByState(state string, offset, limit int) (int64, []model.PlaceTbl) {
	var max int64
	conf.Db.Model(&model.PlaceTbl{}).Where("state = ?", state).Count(&max)
	var places []model.PlaceTbl
	conf.Db.Model(&model.PlaceTbl{}).Where("state = ?", state).Offset(offset).Limit(limit).Find(&places)
	return max, places
}

// SearchPlaces 按字段模糊搜索车位。
func SearchPlaces(field, val string, offset, limit int) ([]model.PlaceTbl, int64) {
	var max int64
	conf.Db.Model(&model.PlaceTbl{}).Where(field+" LIKE ?", "%"+val+"%").Count(&max)
	var places []model.PlaceTbl
	conf.Db.Model(&model.PlaceTbl{}).Where(field+" LIKE ?", "%"+val+"%").Offset(offset).Limit(limit).Find(&places)
	return places, max
}

// DeletePlaceByPlaceid 删除车位。
func DeletePlaceByPlaceid(placeid string) {
	conf.Db.Where("placeid = ?", placeid).Delete(&model.PlaceTbl{})
}

// UserPhoneByOpenid 按 openid 查询手机号。
func UserPhoneByOpenid(openid string) string {
	var phone string
	conf.Db.Model(&model.UserTbl{}).Where("openid = ?", openid).Pluck("phone", &phone)
	return phone
}

// UpdatePlaceStateByPlaceid 更新车位状态，返回受影响行数。
func UpdatePlaceStateByPlaceid(placeid, state string) int64 {
	res := conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Update("state", state)
	return res.RowsAffected
}

// ListParkingSpaces 构建 tab_parking_spaces 过滤查询并分页。
func ListParkingSpaces(name, spacesCode, id, lockID, chargingGunID string, offset, limit int) (int64, []ParkingSpace) {
	query := func() *gorm.DB {
		db := conf.Db.Table("tab_parking_spaces")
		if name != "" {
			db = db.Where("name LIKE ?", "%"+name+"%")
		}
		if spacesCode != "" {
			db = db.Where("spaces_code LIKE ?", "%"+spacesCode+"%")
		}
		if id != "" {
			db = db.Where("id LIKE ?", "%"+id+"%")
		}
		if lockID != "" {
			db = db.Where("lock_id LIKE ?", "%"+lockID+"%")
		}
		if chargingGunID != "" {
			db = db.Where("charging_gun_id LIKE ?", "%"+chargingGunID+"%")
		}
		return db
	}
	var total int64
	query().Count(&total)
	var rows []ParkingSpace
	query().Offset(offset).Limit(limit).Find(&rows)
	return total, rows
}

// UpdateParkingSpaceByID 按 id 更新车位，返回受影响行数。
func UpdateParkingSpaceByID(id interface{}, updates map[string]interface{}) int64 {
	res := conf.Db.Table("tab_parking_spaces").Where("id = ?", id).Updates(updates)
	return res.RowsAffected
}

// DeleteParkingSpaceByID 按 id 删除车位，返回受影响行数。
func DeleteParkingSpaceByID(id interface{}) int64 {
	res := conf.Db.Table("tab_parking_spaces").Where("id = ?", id).Delete(nil)
	return res.RowsAffected
}

// ParkingSpaceStats 统计某车位号下驿享充电的（月/周）电量与电费。
func ParkingSpaceStats(spacesCode string, weekly bool) (float64, decimal.Decimal, bool) {
	var space model.TabParkingSpace
	if err := conf.Db.Where("spaces_code = ?", spacesCode).First(&space).Error; err != nil {
		return 0, decimal.Zero, false
	}
	now := time.Now()
	var start, end time.Time
	if weekly {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = now.AddDate(0, 0, -(weekday - 1))
	} else {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
	if weekly {
		end = start.AddDate(0, 0, 7)
	} else {
		end = start.AddDate(0, 1, 0)
	}
	startStr := start.Format("2006-01-02 15:04:05")
	endStr := end.Format("2006-01-02 15:04:05")

	var degree sql.NullFloat64
	conf.Db.Raw(
		"SELECT COALESCE(SUM(charging_degree),0) FROM order_tbl WHERE spaces_id = ? AND consumption_type = '驿享充电' AND state = '已完成' AND over_time >= ? AND over_time < ?",
		space.ID, startStr, endStr,
	).Row().Scan(&degree)

	season := CurrentSeason()
	feeSQL := "SELECT COALESCE(SUM((e.over_value - e.start_value) / 100 * " +
		"CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END),0) " +
		"FROM tab_order_electronic e " +
		"LEFT JOIN tab_price_time_" + season + " t ON t.id = e.price_time_id " +
		"LEFT JOIN tab_price p ON p.id = t.price_id " +
		"LEFT JOIN order_tbl o ON o.orderid = e.order_id " +
		"LEFT JOIN tab_parking_spaces s ON s.id = o.spaces_id " +
		"WHERE o.spaces_id = ? AND o.consumption_type = '驿享充电' AND o.state = '已完成' AND o.over_time >= ? AND o.over_time < ?"
	var fee decimal.Decimal
	conf.Db.Raw(feeSQL, space.ID, startStr, endStr).Row().Scan(&fee)

	return degree.Float64, fee, true
}
