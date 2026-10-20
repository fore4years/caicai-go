package repository

import (
	"fmt"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// chargeStationGeoCols 充电站查询列（经纬度取自 point 空间字段）。
const chargeStationGeoCols = "id, point, place, name, openid, power_max, pid, create_time, update_time"

// chargeStationDistanceSQL 充电站点到查询点的球面距离（米），对齐 Java ST_Distance_Sphere。
const chargeStationDistanceSQL = "ST_Distance_Sphere(point, ST_GeomFromText(?))"

// ChargingStationByPid 按 pid 查询充电站。
func ChargingStationByPid(pid string) (*model.ChargingStationTbl, error) {
	var s model.ChargingStationTbl
	if err := conf.Db.Raw("SELECT "+chargeStationGeoCols+" FROM charging_station_tbl WHERE pid = ?", pid).Scan(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// ChargingStationByID 按 id 查询充电站。
func ChargingStationByID(id string) (*model.ChargingStationTbl, error) {
	var s model.ChargingStationTbl
	if err := conf.Db.Raw("SELECT "+chargeStationGeoCols+" FROM charging_station_tbl WHERE id = ?", id).Scan(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// ChargingStationsByOpenid 按 openid 查询充电站列表。
func ChargingStationsByOpenid(openid string) []model.ChargingStationTbl {
	var rows []model.ChargingStationTbl
	conf.Db.Raw("SELECT "+chargeStationGeoCols+" FROM charging_station_tbl WHERE openid = ?", openid).Scan(&rows)
	return rows
}

// ChargingStationByPidDB 按 pid 查询充电站（gorm 风格）。
func ChargingStationByPidDB(pid string) (*model.ChargingStationTbl, error) {
	var station model.ChargingStationTbl
	if err := conf.Db.Where("pid = ?", pid).First(&station).Error; err != nil {
		return nil, err
	}
	return &station, nil
}

// SaveChargingStation 保存充电站（ID=0 新增，否则按 id 更新；point 用 ST_GeomFromText 写入）。
func SaveChargingStation(station *model.ChargingStationTbl) error {
	pointWKT := station.Point.String()
	if station.ID == 0 {
		return conf.Db.Exec(
			"INSERT INTO charging_station_tbl (point, place, name, openid, pid, power_max) VALUES (ST_GeomFromText(?),?,?,?,?,?)",
			pointWKT, station.Place, station.Name, station.Openid, station.Pid, station.PowerMax,
		).Error
	}
	return conf.Db.Exec(
		"UPDATE charging_station_tbl SET point = ST_GeomFromText(?), place = ?, name = ?, openid = ?, power_max = ?, update_time = NOW() WHERE id = ?",
		pointWKT, station.Place, station.Name, station.Openid, station.PowerMax, station.ID,
	).Error
}

// CountChargingStations 统计充电站数量。
func CountChargingStations() int64 {
	var total int64
	conf.Db.Model(&model.ChargingStationTbl{}).Count(&total)
	return total
}

// ChargingStationsByPoint 按经纬度距离分页查询充电站。
func ChargingStationsByPoint(lat, lng float64, limit, offset int) []model.ChargingStationTbl {
	var rows []model.ChargingStationTbl
	pointWKT := fmt.Sprintf("POINT(%f %f)", lng, lat)
	conf.Db.Raw(
		"SELECT "+chargeStationGeoCols+", "+chargeStationDistanceSQL+" AS distance FROM charging_station_tbl "+
			"ORDER BY "+chargeStationDistanceSQL+" ASC LIMIT ? OFFSET ?",
		pointWKT, pointWKT, limit, offset,
	).Scan(&rows)
	return rows
}

// AllChargingStations 查询所有充电站。
func AllChargingStations() ([]model.ChargingStationTbl, error) {
	var rows []model.ChargingStationTbl
	tx := conf.Db.Raw("SELECT " + chargeStationGeoCols + " FROM charging_station_tbl").Scan(&rows)
	return rows, tx.Error
}

// InsertReOrderPower 新增二轮车订单充电时段电量。
func InsertReOrderPower(orderID string, meterValue interface{}) error {
	return conf.Db.Exec(
		"INSERT INTO re_order_electronic_tbl (order_id, start_value, over_value, price_time_id) "+
			"VALUES (?,?,?,(SELECT id FROM tab_price_time WHERE start_time < curtime() AND over_time >= curtime() LIMIT 1))",
		orderID, meterValue, meterValue,
	).Error
}

// ReOrderPowerValueByOrderid 查询订单电量差（MAX(over_value)-MIN(start_value)）。
func ReOrderPowerValueByOrderid(orderID string) *decimal.Decimal {
	var d decimal.Decimal
	if err := conf.Db.Raw("SELECT (MAX(over_value)-MIN(start_value)) FROM re_order_electronic_tbl WHERE order_id = ?", orderID).Row().Scan(&d); err != nil {
		return nil
	}
	return &d
}

// ReOrderPowerFeesByOrderid 查询订单电费合计。
func ReOrderPowerFeesByOrderid(orderID string) *decimal.Decimal {
	var d decimal.Decimal
	if err := conf.Db.Raw(`SELECT COALESCE(SUM((over_value - start_value) * (COALESCE(charging_price_tbl.total, 0) + COALESCE(charging_price_tbl.service, 0))),0)
		FROM re_order_electronic_tbl
		LEFT JOIN tab_price_time ON tab_price_time.id = re_order_electronic_tbl.price_time_id
		LEFT JOIN charging_price_tbl ON charging_price_tbl.id = tab_price_time.price_id
		WHERE re_order_electronic_tbl.order_id = ?`, orderID).Row().Scan(&d); err != nil {
		return nil
	}
	return &d
}

// ReOrderPowerElectronicByOrderid 查询订单电费明细（totalFee, serviceFee）。
func ReOrderPowerElectronicByOrderid(orderID string) (decimal.Decimal, decimal.Decimal) {
	var totalFee, serviceFee decimal.Decimal
	if err := conf.Db.Raw(`SELECT
			COALESCE(SUM((over_value - start_value) * COALESCE(charging_price_tbl.total, 0)),0) AS totalFee,
			COALESCE(SUM((over_value - start_value) * COALESCE(charging_price_tbl.service, 0)),0) AS serviceFee
		FROM re_order_electronic_tbl
		LEFT JOIN tab_price_time ON tab_price_time.id = re_order_electronic_tbl.price_time_id
		LEFT JOIN charging_price_tbl ON charging_price_tbl.id = tab_price_time.price_id
		WHERE re_order_electronic_tbl.order_id = ?`, orderID).Row().Scan(&totalFee, &serviceFee); err != nil {
		totalFee, serviceFee = decimal.Zero, decimal.Zero
	}
	return totalFee, serviceFee
}

// ReOrderPowerMaxByOrderid 查询订单最大功率。
func ReOrderPowerMaxByOrderid(orderID string) decimal.Decimal {
	var maxPower decimal.Decimal
	conf.Db.Raw("SELECT COALESCE(max_power,0) FROM re_order_electronic_tbl WHERE order_id = ? LIMIT 1", orderID).Row().Scan(&maxPower)
	return maxPower
}

// ListChargingStates 分页查询充电状态（create_time 倒序）。
func ListChargingStates(productID string, offset, limit int) (int64, []model.ChargingStateTbl) {
	query := func() *gorm.DB {
		db := conf.Db.Model(&model.ChargingStateTbl{})
		if productID != "" {
			db = db.Where("product_id = ?", productID)
		}
		return db.Order("create_time desc")
	}
	var total int64
	query().Count(&total)
	var rows []model.ChargingStateTbl
	query().Offset(offset).Limit(limit).Find(&rows)
	return total, rows
}

// ChargingStationsByPidGorm 按 pid 查询充电站列表（gorm）。
func ChargingStationsByPidGorm(pid string) []model.ChargingStationTbl {
	var rows []model.ChargingStationTbl
	conf.Db.Model(&model.ChargingStationTbl{}).Where("pid = ?", pid).Find(&rows)
	return rows
}

// ProductExistsByPid 判断产品 pid 是否存在。
func ProductExistsByPid(pid string) bool {
	var n int64
	conf.Db.Model(&model.TabProduct{}).Where("pid = ?", pid).Count(&n)
	return n > 0
}

// InsertChargingStation 新增充电站（point 用 ST_GeomFromText 写入）。
func InsertChargingStation(place, name, openid, pid string, powerMax int, longitude, latitude float64) error {
	pointWKT := fmt.Sprintf("POINT(%f %f)", longitude, latitude)
	return conf.Db.Exec(
		"INSERT INTO charging_station_tbl (point, place, name, openid, pid, power_max) VALUES (ST_GeomFromText(?),?,?,?,?,?)",
		pointWKT, place, name, openid, pid, powerMax,
	).Error
}

// DeleteChargingStationByID 删除充电站，返回受影响行数。
func DeleteChargingStationByID(id string) int64 {
	res := conf.Db.Where("id = ?", id).Delete(&model.ChargingStationTbl{})
	return res.RowsAffected
}
