package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// chargingStateDTO 对应 Java domain.ChargingState（charging_state_tbl，驼峰）。
type chargingStateDTO struct {
	ID                   int32          `json:"id"`
	ProductID            string         `json:"productId,omitempty"`
	Direction            string         `json:"direction,omitempty"`
	EmergencyButtonState string         `json:"emergencyButtonState,omitempty"`
	DoorLockState        string         `json:"doorLockState,omitempty"`
	ProductName          string         `json:"productName,omitempty"`
	Model                string         `json:"model,omitempty"`
	Address              string         `json:"address,omitempty"`
	Area                 string         `json:"area,omitempty"`
	AreaOmName           string         `json:"areaOmName,omitempty"`
	AreaOmPhone          string         `json:"areaOmPhone,omitempty"`
	ProductOwnership     string         `json:"productOwnership,omitempty"`
	Phone                string         `json:"phone,omitempty"`
	CreateTime           *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime           *LocalDateTime `json:"updateTime,omitempty"`
}

func chargingStateToDTO(s model.ChargingStateTbl) chargingStateDTO {
	return chargingStateDTO{
		ID:                   s.ID,
		ProductID:            s.ProductID,
		Direction:            s.Direction,
		EmergencyButtonState: s.EmergencyButtonState,
		DoorLockState:        s.DoorLockState,
		ProductName:          s.ProductName,
		Model:                s.Model,
		Address:              s.Address,
		Area:                 s.Area,
		AreaOmName:           s.AreaOmName,
		AreaOmPhone:          s.AreaOmPhone,
		ProductOwnership:     s.ProductOwnership,
		Phone:                s.Phone,
		CreateTime:           timeToLocal(s.CreateTime),
		UpdateTime:           timeToLocal(s.UpdateTime),
	}
}

// chargeStationDTO 对应 Java domain.ChargeStation（charging_station_tbl，驼峰）。
type chargeStationDTO struct {
	ID         int32          `json:"id"`
	Point      string         `json:"point,omitempty"`
	Place      string         `json:"place,omitempty"`
	Name       string         `json:"name,omitempty"`
	Openid     string         `json:"openid,omitempty"`
	Pid        string         `json:"pid,omitempty"`
	PowerMax   string         `json:"powerMax,omitempty"`
	CreateTime *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime *LocalDateTime `json:"updateTime,omitempty"`
}

func chargeStationToDTO(s model.ChargingStationTbl) chargeStationDTO {
	return chargeStationDTO{
		ID:         s.ID,
		Point:      s.Point,
		Place:      s.Place,
		Name:       s.Name,
		Openid:     s.Openid,
		Pid:        s.Pid,
		PowerMax:   strconv.Itoa(int(s.PowerMax)),
		CreateTime: timeToLocal(s.CreateTime),
		UpdateTime: timeToLocal(s.UpdateTime),
	}
}

// chargeStationReq 对应 Java dto.ChargeStationDto。
type chargeStationReq struct {
	Place     string  `json:"place"`
	Name      string  `json:"name"`
	Pid       string  `json:"pid"`
	PowerMax  string  `json:"powerMax"`
	Openid    string  `json:"openid"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// AdminChargeStateController 对齐 Java System.controller.ChargeStateCon。
type AdminChargeStateController struct{}

// List GET /system/chargeState/list?productId=&current=&size=
func (a *AdminChargeStateController) List(c *gin.Context) {
	current, size := parsePage(c)
	productID := c.Query("productId")

	query := func() *gorm.DB {
		db := conf.Db.Model(&model.ChargingStateTbl{})
		if productID != "" {
			db = db.Where("product_id = ?", productID)
		}
		return db
	}

	var total int64
	query().Count(&total)

	var rows []model.ChargingStateTbl
	query().Offset(pageOffset(current, size)).Limit(size).Find(&rows)

	list := make([]chargingStateDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargingStateToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// AdminChargeStationController 对齐 Java System.controller.ChargeStationCon（二轮车充电站）。
type AdminChargeStationController struct{}

// GetStationListByPage GET /system/chargeStation/getStationListByPage
func (a *AdminChargeStationController) GetStationListByPage(c *gin.Context) {
	current, size := parsePage(c)
	var total int64
	conf.Db.Model(&model.ChargingStationTbl{}).Count(&total)
	var rows []model.ChargingStationTbl
	conf.Db.Model(&model.ChargingStationTbl{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]chargeStationDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargeStationToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetStationByPid GET /system/chargeStation/getStationByPid?pid=
func (a *AdminChargeStationController) GetStationByPid(c *gin.Context) {
	var rows []model.ChargingStationTbl
	conf.Db.Model(&model.ChargingStationTbl{}).Where("pid = ?", c.Query("pid")).Find(&rows)
	list := make([]chargeStationDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargeStationToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccess(list))
}

// Add POST /system/chargeStation/add
func (a *AdminChargeStationController) Add(c *gin.Context) {
	var req chargeStationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if req.Pid == "" {
		c.JSON(http.StatusOK, ResultError(400, "产品id不能为空!"))
		return
	}
	var n int64
	conf.Db.Model(&model.TabProduct{}).Where("pid = ?", req.Pid).Count(&n)
	if n > 0 {
		c.JSON(http.StatusOK, ResultError(400, "此产品id已存在，请勿重复添加!"))
		return
	}
	powerMax, _ := strconv.Atoi(req.PowerMax)
	err := conf.Db.Exec(
		"INSERT INTO charging_station_tbl (place, name, openid, pid, power_max) VALUES (?,?,?,?,?)",
		req.Place, req.Name, req.Openid, req.Pid, powerMax,
	).Error
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// UpdateByPid POST /system/chargeStation/updateByPid
func (a *AdminChargeStationController) UpdateByPid(c *gin.Context) {
	var req chargeStationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	powerMax, _ := strconv.Atoi(req.PowerMax)
	res := conf.Db.Model(&model.ChargingStationTbl{}).Where("pid = ?", req.Pid).Updates(map[string]interface{}{
		"name":      req.Name,
		"place":     req.Place,
		"power_max": powerMax,
	})
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// DeleteById DELETE /system/chargeStation/deleteById/:id
func (a *AdminChargeStationController) DeleteById(c *gin.Context) {
	res := conf.Db.Where("id = ?", c.Param("id")).Delete(&model.ChargingStationTbl{})
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// AdminChargeStationFourController 对齐 Java System.controller.ChargeStationFourCon（四轮车，tab_parking_spaces）。
type AdminChargeStationFourController struct{}

// GetStationListByPage GET /system/chargeStationFour/getStationListByPage
func (a *AdminChargeStationFourController) GetStationListByPage(c *gin.Context) {
	current, size := parsePage(c)
	var total int64
	conf.Db.Table("tab_parking_spaces").Count(&total)
	var rows []parkingSpaceDTO
	conf.Db.Table("tab_parking_spaces").Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// GetBySpacesCode GET /system/chargeStationFour/getBySpacesCode?spacesCode=
func (a *AdminChargeStationFourController) GetBySpacesCode(c *gin.Context) {
	var row parkingSpaceDTO
	if err := conf.Db.Table("tab_parking_spaces").Where("spaces_code = ?", c.Query("spacesCode")).Scan(&row).Error; err != nil || row.ID == 0 {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// GetStationById GET /system/chargeStationFour/getStationById?id=
func (a *AdminChargeStationFourController) GetStationById(c *gin.Context) {
	var row parkingSpaceDTO
	if err := conf.Db.Table("tab_parking_spaces").Where("id = ?", c.Query("id")).Scan(&row).Error; err != nil || row.ID == 0 {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// UpdateById POST /system/chargeStationFour/updateById
func (a *AdminChargeStationFourController) UpdateById(c *gin.Context) {
	var req struct {
		ID            int32   `json:"id"`
		Name          string  `json:"name"`
		Place         string  `json:"place"`
		SpacesCode    string  `json:"spacesCode"`
		LockID        string  `json:"lockId"`
		ChargingGunID int32   `json:"chargingGunId"`
		Service       float64 `json:"service"`
		ServiceFee    float64 `json:"serviceFee"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Place != "" {
		updates["place"] = req.Place
	}
	if req.SpacesCode != "" {
		updates["spaces_code"] = req.SpacesCode
	}
	if req.LockID != "" {
		updates["lock_id"] = req.LockID
	}
	if req.ChargingGunID != 0 {
		updates["charging_gun_id"] = req.ChargingGunID
	}
	res := conf.Db.Table("tab_parking_spaces").Where("id = ?", req.ID).Updates(updates)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// DeleteById DELETE /system/chargeStationFour/deleteById/:id
func (a *AdminChargeStationFourController) DeleteById(c *gin.Context) {
	res := conf.Db.Table("tab_parking_spaces").Where("id = ?", c.Param("id")).Delete(nil)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}
