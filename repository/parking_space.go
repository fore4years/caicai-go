package repository

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"caicai-go/conf"
)

// ParkingSpace 对应 Java model.ParkingSpacesBean（驼峰）。
// price_mode / overtime_fee 用 int（DB 值可为 0/1/2），不能用 model 里的 bool。
type ParkingSpace struct {
	Point         string          `gorm:"column:point" json:"point,omitempty"`
	Place         string          `gorm:"column:place" json:"place,omitempty"`
	Name          string          `gorm:"column:name" json:"name,omitempty"`
	Openid        string          `gorm:"column:openid" json:"openid,omitempty"`
	LockID        string          `gorm:"column:lock_id" json:"lockId,omitempty"`
	SpacesCode    string          `gorm:"column:spaces_code" json:"spacesCode,omitempty"`
	OpenTime      string          `gorm:"column:open_time" json:"openTime,omitempty"`
	CloseTime     string          `gorm:"column:close_time" json:"closeTime,omitempty"`
	Service       decimal.Decimal `gorm:"column:service" json:"service,omitempty"`
	ServiceFee    decimal.Decimal `gorm:"column:service_fee" json:"serviceFee,omitempty"`
	State         string          `gorm:"column:state" json:"state,omitempty"`
	OvertimeFee   int             `gorm:"column:overtime_fee" json:"overtimeFee,omitempty"`
	PriceMode     int             `gorm:"column:price_mode" json:"priceMode,omitempty"`
	Distance      float64         `gorm:"column:distance" json:"distance,omitempty"`
	ID            int32           `gorm:"column:id" json:"id"`
	LotID         int32           `gorm:"column:lot_id" json:"lotId,omitempty"`
	ChargingGunID int32           `gorm:"column:charging_gun_id" json:"chargingGunId,omitempty"`
	ImageID       int32           `gorm:"column:image_id" json:"imageId,omitempty"`
	Certificate   int32           `gorm:"column:certificate" json:"certificate,omitempty"`
	Enable        bool            `gorm:"column:enable" json:"enable"`
}

// parkingSpacesGeoCols 与 ParkingSpace 的 gorm column 对齐，用于地理查询（point 用 ST_AsText 输出 WKT）。
const parkingSpacesGeoCols = "tab_parking_spaces.id, ST_AsText(tab_parking_spaces.point) AS point, tab_parking_spaces.place, tab_parking_spaces.name, tab_parking_spaces.openid, tab_parking_spaces.lot_id, tab_parking_spaces.lock_id, tab_parking_spaces.spaces_code, tab_parking_spaces.charging_gun_id, tab_parking_spaces.open_time, tab_parking_spaces.close_time, tab_parking_spaces.image_id, tab_parking_spaces.enable, tab_parking_spaces.certificate, tab_parking_spaces.overtime_fee, tab_parking_spaces.service, tab_parking_spaces.service_fee, tab_parking_spaces.price_mode"

// extractLongitudeLatitude 对齐 Java LocationUtil.extractLongitudeLatitude。
func extractLongitudeLatitude(s string) (lon, lat string) {
	t := strings.TrimSpace(s)
	idx := strings.Index(strings.ToUpper(t), "POINT")
	if idx < 0 {
		return "", ""
	}
	rest := strings.TrimSpace(t[idx+5:])
	rest = strings.TrimPrefix(rest, "(")
	rest = strings.TrimSuffix(rest, ")")
	rest = strings.TrimSpace(strings.ReplaceAll(rest, ",", " "))
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// extractProvinceCityDistrict 对齐 Java LocationUtil.extractProvinceCityDistrict。
func extractProvinceCityDistrict(address string) (province, city, district, street string) {
	if i := strings.Index(address, "省"); i != -1 {
		province = address[:i+len("省")]
		address = address[i+len("省"):]
	}
	if i := strings.Index(address, "市"); i != -1 {
		city = address[:i+len("市")]
		address = address[i+len("市"):]
	}
	if i := strings.Index(address, "区"); i != -1 {
		district = address[:i+len("区")]
		street = address[i+len("区"):]
	} else {
		street = address
	}
	return
}

// NextSpacesCode 取该城市代码下最大车位码 +1。
func NextSpacesCode(table, code string) string {
	var maxCode string
	conf.Db.Raw("SELECT MAX(spaces_code) FROM "+table+" WHERE spaces_code LIKE ?", code+"%").Row().Scan(&maxCode)
	if maxCode == "" {
		maxCode = "0"
	}
	suffix := strings.ReplaceAll(maxCode, code, "")
	n, err := strconv.Atoi(strings.TrimSpace(suffix))
	if err != nil {
		n = 0
	}
	n++
	return code + fmt.Sprintf("%05d", n)
}

// AddPlaceFromSpace 根据车位生成 place_tbl 记录（state=待加锁）。
func AddPlaceFromSpace(spaceID string) bool {
	id, err := strconv.Atoi(spaceID)
	if err != nil {
		return false
	}
	var spacesCode, openid, place, point string
	if err := conf.Db.Raw(
		"SELECT spaces_code, openid, place, ST_AsText(point) FROM tab_parking_spaces WHERE id = ?", id,
	).Row().Scan(&spacesCode, &openid, &place, &point); err != nil || spacesCode == "" {
		return false
	}
	lon, lat := extractLongitudeLatitude(point)
	province, city, district, street := extractProvinceCityDistrict(place)
	res := conf.Db.Exec(
		"INSERT INTO place_tbl(placeid, ownerid, province, city, district, streer, longitude, latitude, state) VALUES (?,?,?,?,?,?,?,?,?)",
		spacesCode, openid, province, city, district, street, lon, lat, "待加锁",
	)
	return res.Error == nil && res.RowsAffected > 0
}

// ParkingSpaceByID 按 id 查询车位。
func ParkingSpaceByID(id string) *ParkingSpace {
	var row ParkingSpace
	if err := conf.Db.Table("tab_parking_spaces").Where("id = ?", id).Scan(&row).Error; err != nil || row.ID == 0 {
		return nil
	}
	return &row
}

// ParkingSpaceByLockID 按 lock_id 查询车位。
func ParkingSpaceByLockID(lockID string) *ParkingSpace {
	var row ParkingSpace
	if err := conf.Db.Table("tab_parking_spaces").Where("lock_id = ?", lockID).Scan(&row).Error; err != nil || row.ID == 0 {
		return nil
	}
	return &row
}

// ParkingSpacesByOpenid 按 openid 查询车位列表。
func ParkingSpacesByOpenid(openid string) []ParkingSpace {
	var rows []ParkingSpace
	conf.Db.Table("tab_parking_spaces").Where("openid = ?", openid).Scan(&rows)
	return rows
}

// InsertParkingSpace 新增车位。
func InsertParkingSpace(wkt, place, name, spacesCode, openid string) error {
	return conf.Db.Exec(
		"INSERT INTO tab_parking_spaces (point, place, name, spaces_code, image_id, openid) VALUES (ST_GeomFromText(?),?,?,?,NULL,?)",
		wkt, place, name, spacesCode, openid,
	).Error
}

// UpdateParkingSpaceEnable 更新车位 enable，返回受影响行数。
func UpdateParkingSpaceEnable(id string, enable bool) int64 {
	res := conf.Db.Table("tab_parking_spaces").Where("id = ?", id).Update("enable", enable)
	return res.RowsAffected
}

// ParkingSpaceDTOBySpacesCode 按 spaces_code 查询车位。
func ParkingSpaceDTOBySpacesCode(code string) *ParkingSpace {
	var row ParkingSpace
	if err := conf.Db.Table("tab_parking_spaces").Where("spaces_code = ?", code).Scan(&row).Error; err != nil || row.ID == 0 {
		return nil
	}
	return &row
}

// ParkingSpaceByPidDirection 按产品与方向查询车位。
func ParkingSpaceByPidDirection(pid, direction string) *ParkingSpace {
	var row ParkingSpace
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+" FROM tab_parking_spaces LEFT JOIN tab_charging_gun g ON g.id = tab_parking_spaces.charging_gun_id "+
			"WHERE g.product_id = ? AND g.direction = ?",
		pid, direction,
	).Scan(&row)
	if row.ID == 0 {
		return nil
	}
	return &row
}

// PageParkingSpacesByPoint 按经纬度距离分页查询车位。
func PageParkingSpacesByPoint(wkt string, offset, limit int) (int64, []ParkingSpace) {
	var total int64
	conf.Db.Table("tab_parking_spaces").Count(&total)
	var rows []ParkingSpace
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+", ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) AS distance FROM tab_parking_spaces "+
			"ORDER BY ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) ASC LIMIT ? OFFSET ?",
		wkt, wkt, limit, offset,
	).Scan(&rows)
	return total, rows
}

// ParkingSpacesByDistance 按距离查询车位。
func ParkingSpacesByDistance(wkt string, dist float64) []ParkingSpace {
	var rows []ParkingSpace
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+", ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) AS distance FROM tab_parking_spaces "+
			"WHERE ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) < ? "+
			"ORDER BY ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) ASC",
		wkt, wkt, dist, wkt,
	).Scan(&rows)
	return rows
}

// SearchParkingSpaces 按名称/地址/编号/距离搜索车位。
func SearchParkingSpaces(key, code, wkt string, dist float64) []ParkingSpace {
	var rows []ParkingSpace
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+", ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) AS distance FROM tab_parking_spaces "+
			"WHERE ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) < ? "+
			"AND (tab_parking_spaces.name LIKE ? OR tab_parking_spaces.place LIKE ?) "+
			"AND tab_parking_spaces.spaces_code LIKE ? ORDER BY distance ASC",
		wkt, wkt, dist, "%"+key+"%", "%"+key+"%", code+"%",
	).Scan(&rows)
	return rows
}

// ParkingSpacesByUserOrderLocation 按用户订单与位置查询车位。
func ParkingSpacesByUserOrderLocation(wkt, openid string, offset, limit int) (int64, []ParkingSpace) {
	var total int64
	conf.Db.Raw(
		"SELECT COUNT(*) FROM tab_parking_spaces LEFT JOIN order_tbl ON (order_tbl.spaces_id = tab_parking_spaces.id AND order_tbl.state <> '已完成')",
	).Row().Scan(&total)

	var rows []ParkingSpace
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+", order_tbl.state AS state, ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) AS distance "+
			"FROM tab_parking_spaces LEFT JOIN order_tbl ON (order_tbl.spaces_id = tab_parking_spaces.id AND order_tbl.state <> '已完成') "+
			"ORDER BY CASE WHEN order_tbl.openid = ? THEN 1 ELSE ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) END LIMIT ? OFFSET ?",
		wkt, openid, wkt, limit, offset,
	).Scan(&rows)

	return total, rows
}

func pageParkingSpaces(db *gorm.DB, offset, limit int) (int64, []ParkingSpace) {
	var total int64
	db.Count(&total)
	var rows []ParkingSpace
	db.Offset(offset).Limit(limit).Scan(&rows)
	return total, rows
}

// PageParkingSpaces 分页查询车位。
func PageParkingSpaces(offset, limit int) (int64, []ParkingSpace) {
	return pageParkingSpaces(conf.Db.Table("tab_parking_spaces"), offset, limit)
}

// PageParkingSpacesByEnable 按 enable 分页查询车位。
func PageParkingSpacesByEnable(enable *bool, offset, limit int) (int64, []ParkingSpace) {
	db := conf.Db.Table("tab_parking_spaces")
	if enable != nil {
		db = db.Where("enable = ?", *enable)
	} else {
		db = db.Where("enable IS NULL")
	}
	return pageParkingSpaces(db, offset, limit)
}

// PageParkingSpacesSearch 按名称/地址模糊分页查询车位。
func PageParkingSpacesSearch(key string, offset, limit int) (int64, []ParkingSpace) {
	db := conf.Db.Table("tab_parking_spaces").Where("name LIKE ? OR place LIKE ?", "%"+key+"%", "%"+key+"%")
	return pageParkingSpaces(db, offset, limit)
}
