package repository

import (
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// PrivatePrice 对应 Java PrivatePriceBean。
type PrivatePrice struct {
	ID      int32           `json:"id"`
	Total   decimal.Decimal `json:"total,omitempty"`
	Name    string          `json:"name,omitempty"`
	Service decimal.Decimal `json:"service,omitempty"`
}

// PrivatePriceTimeRow 时段电价查询行。
type PrivatePriceTimeRow struct {
	StartTime string `gorm:"column:start_time"`
	OverTime  string `gorm:"column:over_time"`
	ID        int32  `gorm:"column:id"`
	PriceID   int32  `gorm:"column:price_id"`
}

const privatePriceTimeColumns = "id, TIME_FORMAT(start_time, '%H:%i:%s') AS start_time, TIME_FORMAT(over_time, '%H:%i:%s') AS over_time, price_id"

// PrivatePriceByID 按 id 查私桩电价。
func PrivatePriceByID(id int32) (*model.PrivateChargingPriceTbl, error) {
	var p model.PrivateChargingPriceTbl
	if err := conf.Db.Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// PrivatePriceTimeRowByID 按 id 查询时段电价行。
func PrivatePriceTimeRowByID(id int) (PrivatePriceTimeRow, error) {
	var row PrivatePriceTimeRow
	if err := conf.Db.Raw("SELECT "+privatePriceTimeColumns+" FROM tab_price_time WHERE id = ?", id).Row().Scan(&row); err != nil {
		return row, err
	}
	return row, nil
}

// UpdatePrivatePriceTime 更新时段电价。
func UpdatePrivatePriceTime(id int32, startTime, overTime string, priceID int32) error {
	return conf.Db.Exec("UPDATE tab_price_time SET start_time = ?, over_time = ?, price_id = ? WHERE id = ?",
		startTime, overTime, priceID, id).Error
}

// InsertPrivatePriceTime 新增时段电价。
func InsertPrivatePriceTime(startTime, overTime string, priceID int32) error {
	return conf.Db.Exec("INSERT INTO tab_price_time (start_time, over_time, price_id) VALUES (?, ?, ?)",
		startTime, overTime, priceID).Error
}

// PrivatePriceTimeRowsAll 分页查询时段电价。
func PrivatePriceTimeRowsAll(offset, limit int) []PrivatePriceTimeRow {
	var rows []PrivatePriceTimeRow
	conf.Db.Raw("SELECT "+privatePriceTimeColumns+" FROM tab_price_time ORDER BY id LIMIT ? OFFSET ?", limit, offset).Scan(&rows)
	return rows
}

// PrivatePriceTimeRowsByTable 按表名查询时段电价。
func PrivatePriceTimeRowsByTable(table string) []PrivatePriceTimeRow {
	var rows []PrivatePriceTimeRow
	conf.Db.Raw("SELECT " + privatePriceTimeColumns + " FROM " + table).Scan(&rows)
	return rows
}

// PrivateChargingFeeByID 查询桩主设置的服务费。
func PrivateChargingFeeByID(id string) *decimal.Decimal {
	var pc model.PrivateChargingTbl
	if err := conf.Db.Where("id = ?", id).First(&pc).Error; err == nil {
		return &pc.Fee
	}
	return nil
}

// PrivatePriceTimeByTime 按时间查最低价时段。
func PrivatePriceTimeByTime(start, end string) (PrivatePriceTimeRow, error) {
	var row PrivatePriceTimeRow
	err := conf.Db.Raw(`
		SELECT pt.id, TIME_FORMAT(pt.start_time, '%H:%i:%s') AS start_time,
		       TIME_FORMAT(pt.over_time, '%H:%i:%s') AS over_time, pt.price_id
		FROM tab_price_time pt
		LEFT JOIN private_charging_price_tbl p ON pt.price_id = p.id
		WHERE pt.over_time >= ? AND pt.start_time <= ?
		ORDER BY (p.total + p.service) LIMIT 1`, start, end).Row().Scan(&row)
	return row, err
}

// PrivatePriceTotalNow 当前时段电价 total。
func PrivatePriceTotalNow(now string) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := conf.Db.Raw(`
		SELECT p.total
		FROM tab_price_time pt
		LEFT JOIN private_charging_price_tbl p ON pt.price_id = p.id
		WHERE pt.start_time < ? AND pt.over_time >= ?
		ORDER BY pt.over_time DESC LIMIT 1`, now, now).Row().Scan(&total)
	return total, err
}
