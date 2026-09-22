package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caicai-go/conf"
)

// ParkingSpacesController 对齐 Java controller.ParkingSpacesController（base /spaces）。
// 地理查询（getByPoint/getByDistance/search 等）暂简化为不分距离的全量/分页返回。
type ParkingSpacesController struct{}

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
	openid := c.PostForm("openid")
	code := c.PostForm("code")
	if openid == "" {
		openid = c.GetString("openid")
	}
	res := conf.Db.Exec("INSERT INTO tab_parking_spaces (name, place, openid, spaces_code) VALUES (?,?,?,?)", name, place, openid, code)
	c.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
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
	}
	spacesPage(c, db)
}

// SetEnableONid GET /spaces/setEnableONid?id=&enable=
func (s *ParkingSpacesController) SetEnableONid(c *gin.Context) {
	enable := c.Query("enable") == "true"
	res := conf.Db.Table("tab_parking_spaces").Where("id = ?", c.Query("id")).Update("enable", enable)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
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
	conf.Db.Table("tab_parking_spaces").
		Joins("JOIN tab_charging_gun g ON g.id = tab_parking_spaces.charging_gun_id").
		Where("g.product_id = ? AND g.direction = ?", c.Query("pid"), c.Query("direction")).
		Scan(&row)
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

// GetByPoint / GetByDistance / Search / GetByUserOrderAndLocation 简化：不分距离
func (s *ParkingSpacesController) GetByPoint(c *gin.Context) {
	spacesPage(c, conf.Db.Table("tab_parking_spaces"))
}

func (s *ParkingSpacesController) GetByDistance(c *gin.Context) {
	var rows []parkingSpaceDTO
	conf.Db.Table("tab_parking_spaces").Limit(100).Scan(&rows)
	c.JSON(http.StatusOK, ResultSuccess(rows))
}

func (s *ParkingSpacesController) Search(c *gin.Context) {
	var rows []parkingSpaceDTO
	conf.Db.Table("tab_parking_spaces").Limit(100).Scan(&rows)
	c.JSON(http.StatusOK, ResultSuccess(rows))
}

func (s *ParkingSpacesController) GetByUserOrderAndLocation(c *gin.Context) {
	spacesPage(c, conf.Db.Table("tab_parking_spaces"))
}

// IsOneLockPerSpot GET /spaces/isOneLockPerSpot?spacesId=
func (s *ParkingSpacesController) IsOneLockPerSpot(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// GetOtherSpacesIdBySpacesId GET /spaces/getOtherSpacesIdBySpacesId?spacesId=
func (s *ParkingSpacesController) GetOtherSpacesIdBySpacesId(c *gin.Context) {
	c.JSON(http.StatusOK, ResultSuccess(0))
}
