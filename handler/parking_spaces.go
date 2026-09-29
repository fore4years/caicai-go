package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caicai-go/conf"
)

// ParkingSpacesController 对齐 Java controller.ParkingSpacesController（base /spaces）。
type ParkingSpacesController struct{}

// parkingSpacesGeoCols 与 parkingSpaceDTO 的 gorm column 对齐，用于地理查询（point 用 ST_AsText 输出 WKT）。
const parkingSpacesGeoCols = "tab_parking_spaces.id, ST_AsText(tab_parking_spaces.point) AS point, tab_parking_spaces.place, tab_parking_spaces.name, tab_parking_spaces.openid, tab_parking_spaces.lot_id, tab_parking_spaces.lock_id, tab_parking_spaces.spaces_code, tab_parking_spaces.charging_gun_id, tab_parking_spaces.open_time, tab_parking_spaces.close_time, tab_parking_spaces.image_id, tab_parking_spaces.enable, tab_parking_spaces.certificate, tab_parking_spaces.overtime_fee, tab_parking_spaces.service, tab_parking_spaces.service_fee, tab_parking_spaces.price_mode"

// wktPoint 构造 ST_GeomFromText 的 WKT 参数（对齐 Java JTS Point "POINT (lon lat)"）。
func wktPoint(lng, lat float64) string {
	return fmt.Sprintf("POINT(%v %v)", lng, lat)
}

// nextSpacesCode 对齐 Java ParkingSpacesServiceImpl.getNextSpacesCode：
// 取该城市代码下最大车位码，去掉前缀后 +1，按 %05d 补齐。
func nextSpacesCode(table, code string) string {
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

// extractLongitudeLatitude 对齐 Java LocationUtil.extractLongitudeLatitude。
// 兼容 JTS "POINT (lon lat)" 与 MySQL ST_AsText "POINT(lon lat)"。
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

// addPlaceFromSpace 对齐 Java ParkingSpacesServiceImpl.addPlace：根据车位生成 place_tbl 记录（state=待加锁）。
func addPlaceFromSpace(spaceID string) bool {
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

func spacesPage(c *gin.Context, db *gorm.DB) {
	current, size := parsePage(c)
	var total int64
	db.Count(&total)
	var rows []parkingSpaceDTO
	db.Offset(pageOffset(current, size)).Limit(size).Scan(&rows)
	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// GetById GET /spaces/getById?id=
func (s *ParkingSpacesController) GetById(c *gin.Context) {
	var row parkingSpaceDTO
	if err := conf.Db.Table("tab_parking_spaces").Where("id = ?", c.Query("id")).Scan(&row).Error; err != nil || row.ID == 0 {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// GetByLock GET /spaces/getByLock?lockId=
func (s *ParkingSpacesController) GetByLock(c *gin.Context) {
	var row parkingSpaceDTO
	if err := conf.Db.Table("tab_parking_spaces").Where("lock_id = ?", c.Query("lockId")).Scan(&row).Error; err != nil || row.ID == 0 {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// GetByOpenid GET /spaces/getByOpenid
func (s *ParkingSpacesController) GetByOpenid(c *gin.Context) {
	var rows []parkingSpaceDTO
	conf.Db.Table("tab_parking_spaces").Where("openid = ?", c.GetString("openid")).Scan(&rows)
	c.JSON(http.StatusOK, ResultSuccess(rows))
}

// Add POST /spaces/add
func (s *ParkingSpacesController) Add(c *gin.Context) {
	name := c.PostForm("name")
	place := c.PostForm("place")
	code := c.PostForm("code")
	latStr := c.PostForm("latitude")
	lngStr := c.PostForm("longitude")
	openid := c.GetString("openid")

	if strings.TrimSpace(code) == "" {
		c.JSON(http.StatusOK, ResultError(400, "城市代码不能为空"))
		return
	}
	spacesCode := nextSpacesCode("tab_parking_spaces", code)

	lat, _ := strconv.ParseFloat(latStr, 64)
	lng, _ := strconv.ParseFloat(lngStr, 64)
	wkt := wktPoint(lng, lat)

	// 证书上传省略（对齐本项目其它 handler 的 COS 省略约定），image_id 传 NULL。
	if err := conf.Db.Exec(
		"INSERT INTO tab_parking_spaces (point, place, name, spaces_code, image_id, openid) VALUES (ST_GeomFromText(?),?,?,?,NULL,?)",
		wkt, place, name, spacesCode, openid,
	).Error; err != nil {
		c.JSON(http.StatusOK, ResultError(400, "添加车位失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// GetAll GET /spaces/getAll
func (s *ParkingSpacesController) GetAll(c *gin.Context) {
	spacesPage(c, conf.Db.Table("tab_parking_spaces"))
}

// GetByEnable GET /spaces/getByEnable?enable=
func (s *ParkingSpacesController) GetByEnable(c *gin.Context) {
	db := conf.Db.Table("tab_parking_spaces")
	if e := c.Query("enable"); e != "" {
		db = db.Where("enable = ?", e == "true")
	} else {
		db = db.Where("enable IS NULL")
	}
	spacesPage(c, db)
}

// SetEnableONid GET /spaces/setEnableONid?id=&enable=
func (s *ParkingSpacesController) SetEnableONid(c *gin.Context) {
	enable := c.Query("enable") == "true"
	res := conf.Db.Table("tab_parking_spaces").Where("id = ?", c.Query("id")).Update("enable", enable)
	b := res.RowsAffected > 0
	if b && enable {
		c.JSON(http.StatusOK, ResultSuccess(addPlaceFromSpace(c.Query("id"))))
		return
	}
	if !enable {
		c.JSON(http.StatusOK, ResultSuccess(true))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(false))
}

// SearchAll GET /spaces/searchAll?key=
func (s *ParkingSpacesController) SearchAll(c *gin.Context) {
	key := c.Query("key")
	db := conf.Db.Table("tab_parking_spaces").Where("name LIKE ? OR place LIKE ?", "%"+key+"%", "%"+key+"%")
	spacesPage(c, db)
}

// GetByPidAndDirection GET /spaces/getByPidAndDirection?pid=&direction=
func (s *ParkingSpacesController) GetByPidAndDirection(c *gin.Context) {
	var row parkingSpaceDTO
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+" FROM tab_parking_spaces LEFT JOIN tab_charging_gun g ON g.id = tab_parking_spaces.charging_gun_id "+
			"WHERE g.product_id = ? AND g.direction = ?",
		c.Query("pid"), c.Query("direction"),
	).Scan(&row)
	if row.ID == 0 {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// GetBySpacesCode GET /spaces/getBySpacesCode?spacesCode=
func (s *ParkingSpacesController) GetBySpacesCode(c *gin.Context) {
	var row parkingSpaceDTO
	if err := conf.Db.Table("tab_parking_spaces").Where("spaces_code = ?", c.Query("spacesCode")).Scan(&row).Error; err != nil || row.ID == 0 {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// GetByPoint GET /spaces/getByPoint?longitude=&latitude=
func (s *ParkingSpacesController) GetByPoint(c *gin.Context) {
	lngStr, latStr := c.Query("longitude"), c.Query("latitude")
	if lngStr == "" || latStr == "" {
		c.JSON(http.StatusOK, ResultError(400, "经纬度不能为空"))
		return
	}
	lng, _ := strconv.ParseFloat(lngStr, 64)
	lat, _ := strconv.ParseFloat(latStr, 64)
	wkt := wktPoint(lng, lat)
	current, size := parsePage(c)

	var total int64
	conf.Db.Table("tab_parking_spaces").Count(&total)

	var rows []parkingSpaceDTO
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+", ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) AS distance FROM tab_parking_spaces "+
			"ORDER BY ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) ASC LIMIT ? OFFSET ?",
		wkt, wkt, size, pageOffset(current, size),
	).Scan(&rows)

	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// GetByDistance GET /spaces/getByDistance?longitude=&latitude=&distance=
func (s *ParkingSpacesController) GetByDistance(c *gin.Context) {
	lngStr, latStr := c.Query("longitude"), c.Query("latitude")
	if lngStr == "" || latStr == "" {
		c.JSON(http.StatusOK, ResultError(400, "经纬度不能为空"))
		return
	}
	distStr := c.Query("distance")
	if distStr == "" {
		c.JSON(http.StatusOK, ResultError(400, "距离不能为空"))
		return
	}
	lng, _ := strconv.ParseFloat(lngStr, 64)
	lat, _ := strconv.ParseFloat(latStr, 64)
	dist, _ := strconv.ParseFloat(distStr, 64)
	wkt := wktPoint(lng, lat)

	var rows []parkingSpaceDTO
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+", ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) AS distance FROM tab_parking_spaces "+
			"WHERE ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) < ? "+
			"ORDER BY ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) ASC",
		wkt, wkt, dist, wkt,
	).Scan(&rows)
	c.JSON(http.StatusOK, ResultSuccess(rows))
}

// Search GET /spaces/search?key=&code=&longitude=&latitude=&distance=
func (s *ParkingSpacesController) Search(c *gin.Context) {
	lngStr, latStr := c.Query("longitude"), c.Query("latitude")
	if lngStr == "" || latStr == "" {
		c.JSON(http.StatusOK, ResultError(400, "经纬度不能为空"))
		return
	}
	distStr := c.Query("distance")
	if distStr == "" {
		c.JSON(http.StatusOK, ResultError(400, "距离不能为空"))
		return
	}
	key := c.Query("key")
	code := c.Query("code")
	lng, _ := strconv.ParseFloat(lngStr, 64)
	lat, _ := strconv.ParseFloat(latStr, 64)
	dist, _ := strconv.ParseFloat(distStr, 64)
	wkt := wktPoint(lng, lat)

	var rows []parkingSpaceDTO
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+", ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) AS distance FROM tab_parking_spaces "+
			"WHERE ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) < ? "+
			"AND (tab_parking_spaces.name LIKE ? OR tab_parking_spaces.place LIKE ?) "+
			"AND tab_parking_spaces.spaces_code LIKE ? ORDER BY distance ASC",
		wkt, wkt, dist, "%"+key+"%", "%"+key+"%", code+"%",
	).Scan(&rows)
	c.JSON(http.StatusOK, ResultSuccess(rows))
}

// GetByUserOrderAndLocation GET /spaces/getByUserOrderAndLocation?openid=&longitude=&latitude=
func (s *ParkingSpacesController) GetByUserOrderAndLocation(c *gin.Context) {
	lngStr, latStr := c.Query("longitude"), c.Query("latitude")
	if lngStr == "" || latStr == "" {
		c.JSON(http.StatusOK, ResultError(400, "经纬度不能为空"))
		return
	}
	openid := c.Query("openid")
	if openid == "" {
		openid = c.GetString("openid")
	}
	lng, _ := strconv.ParseFloat(lngStr, 64)
	lat, _ := strconv.ParseFloat(latStr, 64)
	wkt := wktPoint(lng, lat)
	current, size := parsePage(c)

	var total int64
	conf.Db.Raw(
		"SELECT COUNT(*) FROM tab_parking_spaces LEFT JOIN order_tbl ON (order_tbl.spaces_id = tab_parking_spaces.id AND order_tbl.state <> '已完成')",
	).Row().Scan(&total)

	var rows []parkingSpaceDTO
	conf.Db.Raw(
		"SELECT "+parkingSpacesGeoCols+", order_tbl.state AS state, ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) AS distance "+
			"FROM tab_parking_spaces LEFT JOIN order_tbl ON (order_tbl.spaces_id = tab_parking_spaces.id AND order_tbl.state <> '已完成') "+
			"ORDER BY CASE WHEN order_tbl.openid = ? THEN 1 ELSE ST_Distance_Sphere(tab_parking_spaces.point, ST_GeomFromText(?)) END LIMIT ? OFFSET ?",
		wkt, openid, wkt, size, pageOffset(current, size),
	).Scan(&rows)

	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// IsOneLockPerSpot GET /spaces/isOneLockPerSpot?spacesId=
func (s *ParkingSpacesController) IsOneLockPerSpot(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("spacesId"))
	c.JSON(http.StatusOK, ResultSuccess(isOneLockPerSpot(int32(id))))
}

// GetOtherSpacesIdBySpacesId GET /spaces/getOtherSpacesIdBySpacesId?spacesId=
func (s *ParkingSpacesController) GetOtherSpacesIdBySpacesId(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("spacesId"))
	c.JSON(http.StatusOK, ResultSuccess(otherSpacesIDBySpacesID(int32(id))))
}
