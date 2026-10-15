package repository

import (
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// chargeStationGeoCols 充电站查询列（经纬度取 longitude/latitude）。
const chargeStationGeoCols = "id, longitude, latitude, place, name, openid, power_max, pid, create_time, update_time"

// chargeStationDistanceSQL 充电站经纬度到查询点的球面距离（米），Haversine 公式。
const chargeStationDistanceSQL = "(6371000 * 2 * ASIN(SQRT(POWER(SIN((RADIANS(latitude) - RADIANS(?)) / 2), 2) + COS(RADIANS(?)) * COS(RADIANS(latitude)) * POWER(SIN((RADIANS(longitude) - RADIANS(?)) / 2), 2))))"

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

// SaveChargingStation 保存充电站（存在则更新）。
func SaveChargingStation(station *model.ChargingStationTbl) error {
	return conf.Db.Save(station).Error
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
	conf.Db.Raw(
		"SELECT "+chargeStationGeoCols+", "+chargeStationDistanceSQL+" AS distance FROM charging_station_tbl "+
			"ORDER BY "+chargeStationDistanceSQL+" ASC LIMIT ? OFFSET ?",
		lat, lat, lng, lat, lat, lng, limit, offset,
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
			"VALUES (?,?,?,(SELECT id FROM tab_price_time WHERE start_time < curtime() AND over_time >= curtime()))",
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
	if err := conf.Db.Raw(`SELECT SUM((over_value - start_value) * (charging_price_tbl.total + charging_price_tbl.service))
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
			SUM((over_value - start_value) * charging_price_tbl.total) AS totalFee,
			SUM((over_value - start_value) * charging_price_tbl.service) AS serviceFee
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
