package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"caicai-go/repository"
)

// ParkingSpacesController 对齐 Java controller.ParkingSpacesController（base /spaces）。
type ParkingSpacesController struct{}

// wktPoint 构造 ST_GeomFromText 的 WKT 参数（对齐 Java JTS Point "POINT (lon lat)"）。
func wktPoint(lng, lat float64) string {
	return fmt.Sprintf("POINT(%v %v)", lng, lat)
}

// GetById GET /spaces/getById?id=
func (s *ParkingSpacesController) GetById(c *gin.Context) {
	row := repository.ParkingSpaceByID(c.Query("id"))
	if row == nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// GetByLock GET /spaces/getByLock?lockId=
func (s *ParkingSpacesController) GetByLock(c *gin.Context) {
	row := repository.ParkingSpaceByLockID(c.Query("lockId"))
	if row == nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// GetByOpenid GET /spaces/getByOpenid
func (s *ParkingSpacesController) GetByOpenid(c *gin.Context) {
	rows := repository.ParkingSpacesByOpenid(c.GetString("openid"))
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
	spacesCode := repository.NextSpacesCode("tab_parking_spaces", code)

	lat, _ := strconv.ParseFloat(latStr, 64)
	lng, _ := strconv.ParseFloat(lngStr, 64)
	wkt := wktPoint(lng, lat)

	// 证书上传省略（对齐本项目其它 handler 的 COS 省略约定），image_id 传 NULL。
	if err := repository.InsertParkingSpace(wkt, place, name, spacesCode, openid); err != nil {
		c.JSON(http.StatusOK, ResultError(400, "添加车位失败"))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// GetAll GET /spaces/getAll
func (s *ParkingSpacesController) GetAll(c *gin.Context) {
	current, size := parsePage(c)
	total, rows := repository.PageParkingSpaces(pageOffset(current, size), size)
	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// GetByEnable GET /spaces/getByEnable?enable=
func (s *ParkingSpacesController) GetByEnable(c *gin.Context) {
	current, size := parsePage(c)
	var enable *bool
	if e := c.Query("enable"); e != "" {
		v := e == "true"
		enable = &v
	}
	total, rows := repository.PageParkingSpacesByEnable(enable, pageOffset(current, size), size)
	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// SetEnableONid GET /spaces/setEnableONid?id=&enable=
func (s *ParkingSpacesController) SetEnableONid(c *gin.Context) {
	enable := c.Query("enable") == "true"
	b := repository.UpdateParkingSpaceEnable(c.Query("id"), enable) > 0
	if b && enable {
		c.JSON(http.StatusOK, ResultSuccess(repository.AddPlaceFromSpace(c.Query("id"))))
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
	current, size := parsePage(c)
	total, rows := repository.PageParkingSpacesSearch(key, pageOffset(current, size), size)
	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// GetByPidAndDirection GET /spaces/getByPidAndDirection?pid=&direction=
func (s *ParkingSpacesController) GetByPidAndDirection(c *gin.Context) {
	row := repository.ParkingSpaceByPidDirection(c.Query("pid"), c.Query("direction"))
	if row == nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// GetBySpacesCode GET /spaces/getBySpacesCode?spacesCode=
func (s *ParkingSpacesController) GetBySpacesCode(c *gin.Context) {
	row := repository.ParkingSpaceDTOBySpacesCode(c.Query("spacesCode"))
	if row == nil {
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

	total, rows := repository.PageParkingSpacesByPoint(wkt, pageOffset(current, size), size)

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

	rows := repository.ParkingSpacesByDistance(wkt, dist)
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

	rows := repository.SearchParkingSpaces(key, code, wkt, dist)
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

	total, rows := repository.ParkingSpacesByUserOrderLocation(wkt, openid, pageOffset(current, size), size)

	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// IsOneLockPerSpot GET /spaces/repository.IsOneLockPerSpot?spacesId=
func (s *ParkingSpacesController) IsOneLockPerSpot(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("spacesId"))
	c.JSON(http.StatusOK, ResultSuccess(repository.IsOneLockPerSpot(int32(id))))
}

// GetOtherSpacesIdBySpacesId GET /spaces/getOtherSpacesIdBySpacesId?spacesId=
func (s *ParkingSpacesController) GetOtherSpacesIdBySpacesId(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("spacesId"))
	c.JSON(http.StatusOK, ResultSuccess(repository.OtherSpacesIDBySpacesID(int32(id))))
}
