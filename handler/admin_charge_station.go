package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/model"
	"caicai-go/repository"
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
	Longitude  float64        `json:"longitude,omitempty"`
	Latitude   float64        `json:"latitude,omitempty"`
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
		Longitude:  s.Point.Lng(),
		Latitude:   s.Point.Lat(),
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

	total, rows := repository.ListChargingStates(productID, pageOffset(current, size), size)

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
	repository.Count(&model.ChargingStationTbl{}, &total)
	var rows []model.ChargingStationTbl
	repository.Page(&model.ChargingStationTbl{}, "", pageOffset(current, size), size, &rows)
	list := make([]chargeStationDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargeStationToDTO(r))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetStationByPid GET /system/chargeStation/getStationByPid?pid=
func (a *AdminChargeStationController) GetStationByPid(c *gin.Context) {
	rows := repository.ChargingStationsByPidGorm(c.Query("pid"))
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
	if repository.ProductExistsByPid(req.Pid) {
		c.JSON(http.StatusOK, ResultError(400, "此产品id已存在，请勿重复添加!"))
		return
	}
	powerMax, _ := strconv.Atoi(req.PowerMax)
	err := repository.InsertChargingStation(req.Place, req.Name, req.Openid, req.Pid, powerMax, req.Longitude, req.Latitude)
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
	station, err := repository.ChargingStationByPid(req.Pid)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	station.Name = req.Name
	station.Place = req.Place
	if powerMax, err := strconv.Atoi(req.PowerMax); err == nil {
		station.PowerMax = int32(powerMax)
	}
	if req.Longitude != 0 || req.Latitude != 0 {
		station.Point = model.NewPoint(req.Longitude, req.Latitude)
	}
	if err := repository.SaveChargingStation(station); err != nil {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// DeleteById DELETE /system/chargeStation/deleteById/:id
func (a *AdminChargeStationController) DeleteById(c *gin.Context) {
	rowsAffected := repository.DeleteChargingStationByID(c.Param("id"))
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}

// AdminChargeStationFourController 对齐 Java System.controller.ChargeStationFourCon（四轮车，tab_parking_spaces）。
type AdminChargeStationFourController struct{}

// GetStationListByPage GET /system/chargeStationFour/getStationListByPage
func (a *AdminChargeStationFourController) GetStationListByPage(c *gin.Context) {
	current, size := parsePage(c)
	total, rows := repository.PageParkingSpaces(pageOffset(current, size), size)
	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// GetBySpacesCode GET /system/chargeStationFour/getBySpacesCode?spacesCode=
func (a *AdminChargeStationFourController) GetBySpacesCode(c *gin.Context) {
	row := repository.ParkingSpaceDTOBySpacesCode(c.Query("spacesCode"))
	if row == nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// GetStationById GET /system/chargeStationFour/getStationById?id=
func (a *AdminChargeStationFourController) GetStationById(c *gin.Context) {
	row := repository.ParkingSpaceByID(c.Query("id"))
	if row == nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(row))
}

// UpdateById POST /system/chargeStationFour/updateById
func (a *AdminChargeStationFourController) UpdateById(c *gin.Context) {
	var req struct {
		ID            int32           `json:"id"`
		Name          string          `json:"name"`
		Place         string          `json:"place"`
		SpacesCode    string          `json:"spacesCode"`
		LockID        string          `json:"lockId"`
		ChargingGunID int32           `json:"chargingGunId"`
		Service       decimal.Decimal `json:"service"`
		ServiceFee    decimal.Decimal `json:"serviceFee"`
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
	if !req.Service.IsZero() {
		updates["service"] = req.Service
	}
	if !req.ServiceFee.IsZero() {
		updates["service_fee"] = req.ServiceFee
	}
	rowsAffected := repository.UpdateParkingSpaceByID(req.ID, updates)
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}

// DeleteById DELETE /system/chargeStationFour/deleteById/:id
func (a *AdminChargeStationFourController) DeleteById(c *gin.Context) {
	rowsAffected := repository.DeleteParkingSpaceByID(c.Param("id"))
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}
