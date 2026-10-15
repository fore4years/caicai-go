package repository

import (
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// PriceDTO 对应 Java ChargingPriceBean / PriceBean（id/total/name/service）。
type PriceDTO struct {
	ID      int32           `json:"id,omitempty"`
	Total   decimal.Decimal `json:"total,omitempty"`
	Name    string          `json:"name,omitempty"`
	Service decimal.Decimal `json:"service,omitempty"`
}

// PriceTimeRow 用于扫描 tab_price_time 的行（时间列转 HH:mm:ss 字符串）。
type PriceTimeRow struct {
	StartTime string
	OverTime  string
	ID        int32
	PriceID   int32
}

// ChargingPriceByID 按主键查询 charging_price_tbl。
func ChargingPriceByID(id interface{}) (*model.ChargingPriceTbl, error) {
	var p model.ChargingPriceTbl
	if err := conf.Db.Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// TabPriceByID 按主键查询 tab_price。
func TabPriceByID(id interface{}) (*model.TabPrice, error) {
	var p model.TabPrice
	if err := conf.Db.Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// LowestChargingPrice 按 total 升序取第一条。
func LowestChargingPrice() (*model.ChargingPriceTbl, error) {
	var p model.ChargingPriceTbl
	if err := conf.Db.Order("total asc").First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// LowestTabPrice 按 total 升序取第一条。
func LowestTabPrice() (*model.TabPrice, error) {
	var p model.TabPrice
	if err := conf.Db.Order("total asc").First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// InsertChargingPrice 插入 charging_price_tbl（total/name/service）。
func InsertChargingPrice(v *model.ChargingPriceTbl) error {
	return conf.Db.Select("total", "name", "service").Create(v).Error
}

// InsertTabPrice 插入 tab_price（total/name/service）。
func InsertTabPrice(v *model.TabPrice) error {
	return conf.Db.Select("total", "name", "service").Create(v).Error
}

// InsertTabPriceTime 插入 tab_price_time。
func InsertTabPriceTime(startTime, overTime string, priceID int32) error {
	return conf.Db.Exec(
		"INSERT INTO tab_price_time (start_time, over_time, price_id) VALUES (?, ?, ?)",
		startTime, overTime, priceID,
	).Error
}

// ChargingPriceByTime 按时间查最低电价（tab_price_time LEFT JOIN charging_price_tbl）。
func ChargingPriceByTime(start, end string) []PriceDTO {
	var rows []PriceDTO
	conf.Db.Raw(
		`SELECT charging_price_tbl.id, charging_price_tbl.total, charging_price_tbl.name, charging_price_tbl.service
		 FROM tab_price_time
		 LEFT JOIN charging_price_tbl ON tab_price_time.price_id = charging_price_tbl.id
		 WHERE tab_price_time.over_time >= ? AND tab_price_time.start_time <= ?
		 ORDER BY (charging_price_tbl.total + charging_price_tbl.service) LIMIT 1`,
		start, end,
	).Scan(&rows)
	return rows
}

// TabPriceByTime 按时间查最低电价（tab_price_time LEFT JOIN tab_price）。
func TabPriceByTime(start, end string) []PriceDTO {
	var rows []PriceDTO
	conf.Db.Raw(
		`SELECT tab_price.id, tab_price.total, tab_price.name, tab_price.service
		 FROM tab_price_time
		 LEFT JOIN tab_price ON tab_price_time.price_id = tab_price.id
		 WHERE tab_price_time.over_time >= ? AND tab_price_time.start_time <= ?
		 ORDER BY (tab_price.total + tab_price.service) LIMIT 1`,
		start, end,
	).Scan(&rows)
	return rows
}

// PriceTimeRowByID 按 id 查询 tab_price_time 行。
func PriceTimeRowByID(id interface{}) []PriceTimeRow {
	var rows []PriceTimeRow
	conf.Db.Raw(
		`SELECT id, TIME_FORMAT(start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(over_time,'%H:%i:%s') AS over_time, price_id
		 FROM tab_price_time WHERE id = ?`, id,
	).Scan(&rows)
	return rows
}

// PriceTimeRowsAll 分页查询 tab_price_time 行。
func PriceTimeRowsAll(offset, limit int) []PriceTimeRow {
	var rows []PriceTimeRow
	conf.Db.Raw(
		`SELECT id, TIME_FORMAT(start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(over_time,'%H:%i:%s') AS over_time, price_id
		 FROM tab_price_time LIMIT ? OFFSET ?`,
		limit, offset,
	).Scan(&rows)
	return rows
}

// ChargingPriceTimeByTime 按时间查最低电价时段。
func ChargingPriceTimeByTime(start, end string) []PriceTimeRow {
	var rows []PriceTimeRow
	conf.Db.Raw(
		`SELECT tab_price_time.id, TIME_FORMAT(tab_price_time.start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(tab_price_time.over_time,'%H:%i:%s') AS over_time, tab_price_time.price_id
		 FROM tab_price_time
		 LEFT JOIN charging_price_tbl ON tab_price_time.price_id = charging_price_tbl.id
		 WHERE tab_price_time.over_time >= ? AND tab_price_time.start_time <= ?
		 ORDER BY (charging_price_tbl.total + charging_price_tbl.service) LIMIT 1`,
		start, end,
	).Scan(&rows)
	return rows
}

// ChargingPriceTotalNow 当前时段 charging_price_tbl.total。
func ChargingPriceTotalNow() []decimal.Decimal {
	var totals []decimal.Decimal
	conf.Db.Raw(
		`SELECT charging_price_tbl.total
		 FROM tab_price_time
		 LEFT JOIN charging_price_tbl ON tab_price_time.price_id = charging_price_tbl.id
		 WHERE tab_price_time.start_time < CURTIME() AND tab_price_time.over_time >= CURTIME()
		 ORDER BY tab_price_time.over_time DESC LIMIT 1`,
	).Scan(&totals)
	return totals
}

// TabPriceTimeByTime 按时间查最低电价时段（tab_price）。
func TabPriceTimeByTime(start, end string) []PriceTimeRow {
	var rows []PriceTimeRow
	conf.Db.Raw(
		`SELECT tab_price_time.id, TIME_FORMAT(tab_price_time.start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(tab_price_time.over_time,'%H:%i:%s') AS over_time, tab_price_time.price_id
		 FROM tab_price_time
		 LEFT JOIN tab_price ON tab_price_time.price_id = tab_price.id
		 WHERE tab_price_time.over_time >= ? AND tab_price_time.start_time <= ?
		 ORDER BY (tab_price.total + tab_price.service) LIMIT 1`,
		start, end,
	).Scan(&rows)
	return rows
}

// TabPriceNow 当前时段 tab_price 的 total/service。
func TabPriceNow() (decimal.Decimal, decimal.Decimal, bool) {
	type row struct {
		Total   decimal.Decimal
		Service decimal.Decimal
	}
	var rows []row
	err := conf.Db.Raw(
		`SELECT tab_price.total, tab_price.service
		 FROM tab_price_time
		 LEFT JOIN tab_price ON tab_price_time.price_id = tab_price.id
		 WHERE tab_price_time.start_time < CURTIME() AND tab_price_time.over_time >= CURTIME()
		 ORDER BY tab_price_time.over_time DESC LIMIT 1`,
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return decimal.Zero, decimal.Zero, false
	}
	return rows[0].Total, rows[0].Service, true
}

// TabPriceAllNow 当前时段 tab_price 的嵌套电价。
func TabPriceAllNow() []PriceDTO {
	var rows []PriceDTO
	conf.Db.Raw(
		`SELECT tab_price.id, tab_price.total, tab_price.name, tab_price.service
		 FROM tab_price_time
		 LEFT JOIN tab_price ON tab_price_time.price_id = tab_price.id
		 WHERE tab_price_time.start_time < CURTIME() AND tab_price_time.over_time >= CURTIME()
		 ORDER BY tab_price_time.over_time DESC LIMIT 1`,
	).Scan(&rows)
	return rows
}
