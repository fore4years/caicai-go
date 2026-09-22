package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ DTO（驼峰，对齐 Java Bean 序列化） ============

// chargingGunDTO 对应 Java model.ChargingGunBean（tab_charging_gun）。
type chargingGunDTO struct {
	ID         int32          `json:"id"`
	Name       string         `json:"name,omitempty"`
	UnitPrice  float64        `json:"unitPrice,omitempty"`
	MeterValue int32          `json:"meterValue,omitempty"`
	State      string         `json:"state,omitempty"`
	ProductID  string         `json:"productId,omitempty"`
	Direction  string         `json:"direction,omitempty"`
	Model      int32          `json:"model,omitempty"`
	Pwm        int32          `json:"pwm,omitempty"`
	UpdateTime *LocalDateTime `json:"updateTime,omitempty"`
}

func chargingGunToDTO(g model.TabChargingGun) chargingGunDTO {
	return chargingGunDTO{
		ID:         g.ID,
		Name:       g.Name,
		UnitPrice:  g.UnitPrice,
		MeterValue: g.MeterValue,
		State:      g.State,
		ProductID:  g.ProductID,
		Direction:  g.Direction,
		Model:      g.Model,
		Pwm:        g.Pwm,
		UpdateTime: timeToLocal(g.UpdateTime),
	}
}

// stationManagementDTO 对应 Java domain.ChargingStationManagement。
type stationManagementDTO struct {
	ID         int32          `json:"id"`
	Openid     string         `json:"openid,omitempty"`
	Name       string         `json:"name,omitempty"`
	Place      string         `json:"place,omitempty"`
	ParkingNum int32          `json:"parkingNum,omitempty"`
	ProductID  string         `json:"productId,omitempty"`
	Phone      string         `json:"phone,omitempty"`
	Email      string         `json:"email,omitempty"`
	Reviser    string         `json:"reviser,omitempty"`
	CreateTime *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime *LocalDateTime `json:"updateTime,omitempty"`
}

func stationManagementToDTO(m model.ChargingStationManagementTbl) stationManagementDTO {
	return stationManagementDTO{
		ID:         m.ID,
		Openid:     m.Openid,
		Name:       m.Name,
		Place:      m.Place,
		ParkingNum: m.ParkingNum,
		ProductID:  m.ProductID,
		Phone:      m.Phone,
		Email:      m.Email,
		Reviser:    m.Reviser,
		CreateTime: timeToLocal(m.CreateTime),
		UpdateTime: timeToLocal(m.UpdateTime),
	}
}

// ledDTO 对应 Java model.LedBean（led_tbl）。
type ledDTO struct {
	ID         int32          `json:"id"`
	Imei       string         `json:"imei,omitempty"`
	GatewayID  string         `json:"gatewayId,omitempty"`
	CreateTime *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime *LocalDateTime `json:"updateTime,omitempty"`
}

func ledToDTO(l model.LedTbl) ledDTO {
	return ledDTO{
		ID:         l.ID,
		Imei:       l.Imei,
		GatewayID:  l.GatewayID,
		CreateTime: timeToLocal(l.CreateTime),
		UpdateTime: timeToLocal(l.UpdateTime),
	}
}

// receptacleDTO 对应 Java domain.Receptacle（receptacle_tbl）。
type receptacleDTO struct {
	ID         int32          `json:"id"`
	Name       string         `json:"name,omitempty"`
	Number     int32          `json:"number,omitempty"`
	Status     int32          `json:"status,omitempty"`
	Pid        string         `json:"pid,omitempty"`
	CreateTime *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime *LocalDateTime `json:"updateTime,omitempty"`
}

func receptacleToDTO(r model.ReceptacleTbl) receptacleDTO {
	return receptacleDTO{
		ID:         r.ID,
		Name:       r.Name,
		Number:     r.Number,
		Status:     r.Status,
		Pid:        r.Pid,
		CreateTime: timeToLocal(r.CreateTime),
		UpdateTime: timeToLocal(r.UpdateTime),
	}
}

// imeiDTO 对应 Java domain.ImeiEntity（imei_tab）。
type imeiDTO struct {
	ID              int64          `json:"id"`
	ChargingStation string         `json:"chargingStation,omitempty"`
	ChargingSocket  string         `json:"chargingSocket,omitempty"`
	Status          int32          `json:"status,omitempty"`
	CreatedTime     *LocalDateTime `json:"createdTime,omitempty"`
	UpdatedTime     *LocalDateTime `json:"updatedTime,omitempty"`
	Deleted         int32          `json:"deleted,omitempty"`
}

func imeiToDTO(i model.ImeiTab) imeiDTO {
	return imeiDTO{
		ID:              i.ID,
		ChargingStation: i.ChargingStation,
		ChargingSocket:  i.ChargingSocket,
		Status:          i.Status,
		CreatedTime:     timeToLocal(i.CreatedTime),
		UpdatedTime:     timeToLocal(i.UpdatedTime),
		Deleted:         i.Deleted,
	}
}

// ============ GatewayController（/gateway） ============

type GatewayController struct{}

func (c *GatewayController) GetById(ctx *gin.Context) {
	var g model.TabGateway
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&g).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(gatewayToDTO(g)))
}

func (c *GatewayController) Add(ctx *gin.Context) {
	var g model.TabGateway
	if err := ctx.ShouldBindJSON(&g); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&g).Error == nil))
}

func (c *GatewayController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.TabGateway{}).Count(&total)
	var rows []model.TabGateway
	conf.Db.Model(&model.TabGateway{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]gatewayDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, gatewayToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// ============ LockController（/lock） ============

type LockController struct{}

func (c *LockController) GetById(ctx *gin.Context) {
	var l model.LockTbl
	if err := conf.Db.Where("lockid = ?", ctx.Query("id")).First(&l).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(lockToDTO(l)))
}

func (c *LockController) GetByLockId(ctx *gin.Context) {
	var l model.LockTbl
	if err := conf.Db.Where("lockid = ?", ctx.Query("lockId")).First(&l).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(lockToDTO(l)))
}

func (c *LockController) Add(ctx *gin.Context) {
	var l model.LockTbl
	if err := ctx.ShouldBindJSON(&l); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&l).Error == nil))
}

func (c *LockController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.LockTbl{}).Count(&total)
	var rows []model.LockTbl
	conf.Db.Model(&model.LockTbl{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]lockDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, lockToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (c *LockController) GetByPid(ctx *gin.Context) {
	var l model.LockTbl
	if err := conf.Db.Where("product_id = ?", ctx.Query("pid")).First(&l).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(lockToDTO(l)))
}

func (c *LockController) GetByOrder(ctx *gin.Context) {
	var l model.LockTbl
	err := conf.Db.Raw(
		"SELECT l.* FROM order_tbl o LEFT JOIN tab_parking_spaces ps ON o.spaces_id = ps.id LEFT JOIN lock_tbl l ON ps.lock_id = l.lockid WHERE o.orderid = ?",
		ctx.Query("orderid"),
	).Scan(&l).Error
	if err != nil || l.Lockid == "" {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(lockToDTO(l)))
}

// ============ ChargingGunController（/chargingGun） ============

type ChargingGunController struct{}

func (c *ChargingGunController) GetById(ctx *gin.Context) {
	var g model.TabChargingGun
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&g).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargingGunToDTO(g)))
}

func (c *ChargingGunController) Add(ctx *gin.Context) {
	var g model.TabChargingGun
	if err := ctx.ShouldBindJSON(&g); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&g).Error == nil))
}

func (c *ChargingGunController) AddPrivateGun(ctx *gin.Context) {
	var g model.TabChargingGun
	if err := ctx.ShouldBindJSON(&g); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&g).Error == nil))
}

func (c *ChargingGunController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.TabChargingGun{}).Count(&total)
	var rows []model.TabChargingGun
	conf.Db.Model(&model.TabChargingGun{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]chargingGunDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargingGunToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (c *ChargingGunController) GetByPidAndDirection(ctx *gin.Context) {
	var g model.TabChargingGun
	if err := conf.Db.Where("product_id = ? AND direction = ?", ctx.Query("pid"), ctx.Query("direction")).First(&g).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargingGunToDTO(g)))
}

func (c *ChargingGunController) UpdatePwm(ctx *gin.Context) {
	pwm, _ := strconv.Atoi(ctx.Query("pwm"))
	res := conf.Db.Model(&model.TabChargingGun{}).
		Where("product_id = ? AND direction = ?", ctx.Query("product_id"), ctx.Query("direction")).
		Update("pwm", pwm)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ ChargingStateController（/chargingState） ============

type ChargingStateController struct{}

func (c *ChargingStateController) List(ctx *gin.Context) {
	var rows []model.ChargingStateTbl
	conf.Db.Model(&model.ChargingStateTbl{}).Find(&rows)
	list := make([]chargingStateDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargingStateToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *ChargingStateController) Page(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.ChargingStateTbl{}).Count(&total)
	var rows []model.ChargingStateTbl
	conf.Db.Model(&model.ChargingStateTbl{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]chargingStateDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargingStateToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (c *ChargingStateController) GetById(ctx *gin.Context) {
	var s model.ChargingStateTbl
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&s).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargingStateToDTO(s)))
}

func (c *ChargingStateController) Save(ctx *gin.Context) {
	var s model.ChargingStateTbl
	if err := ctx.ShouldBindJSON(&s); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&s).Error == nil))
}

func (c *ChargingStateController) Update(ctx *gin.Context) {
	var s model.ChargingStateTbl
	if err := ctx.ShouldBindJSON(&s); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Model(&model.ChargingStateTbl{}).Where("id = ?", s.ID).Updates(s)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *ChargingStateController) DeleteById(ctx *gin.Context) {
	res := conf.Db.Where("id = ?", ctx.Query("id")).Delete(&model.ChargingStateTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ StationManagementController（/chargingStations/management） ============

type StationManagementController struct{}

func (c *StationManagementController) GetAll(ctx *gin.Context) {
	var rows []model.ChargingStationManagementTbl
	conf.Db.Model(&model.ChargingStationManagementTbl{}).Find(&rows)
	list := make([]stationManagementDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, stationManagementToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *StationManagementController) GetById(ctx *gin.Context) {
	var m model.ChargingStationManagementTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&m).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(stationManagementToDTO(m)))
}

func (c *StationManagementController) GetByOpenId(ctx *gin.Context) {
	var rows []model.ChargingStationManagementTbl
	conf.Db.Where("openid = ?", ctx.Param("openid")).Find(&rows)
	list := make([]stationManagementDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, stationManagementToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *StationManagementController) Save(ctx *gin.Context) {
	var m model.ChargingStationManagementTbl
	if err := ctx.ShouldBindJSON(&m); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&m).Error == nil))
}

func (c *StationManagementController) Update(ctx *gin.Context) {
	var m model.ChargingStationManagementTbl
	if err := ctx.ShouldBindJSON(&m); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if id, err := strconv.Atoi(ctx.Param("id")); err == nil {
		m.ID = int32(id)
	}
	res := conf.Db.Model(&model.ChargingStationManagementTbl{}).Where("id = ?", m.ID).Updates(m)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *StationManagementController) Delete(ctx *gin.Context) {
	res := conf.Db.Where("id = ?", ctx.Param("id")).Delete(&model.ChargingStationManagementTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ LedController（/led） ============

type LedController struct{}

func (c *LedController) GetById(ctx *gin.Context) {
	var l model.LedTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&l).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(ledToDTO(l)))
}

// ============ ReceptacleController（/receptacle） ============

type ReceptacleController struct{}

func (c *ReceptacleController) GetReceptacleAll(ctx *gin.Context) {
	var rows []model.ReceptacleTbl
	conf.Db.Where("pid = ?", ctx.Query("pid")).Find(&rows)
	list := make([]receptacleDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, receptacleToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *ReceptacleController) Add(ctx *gin.Context) {
	var r model.ReceptacleTbl
	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&r).Error == nil))
}

func (c *ReceptacleController) BatchAdd(ctx *gin.Context) {
	// body {pid, count?} 简化为：批量插入 count 个插座
	var req struct {
		Pid   string `json:"pid"`
		Count int    `json:"count"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Count <= 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	for i := 1; i <= req.Count; i++ {
		conf.Db.Create(&model.ReceptacleTbl{Pid: req.Pid, Number: int32(i), Status: 0, Name: strconv.Itoa(i)})
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

func (c *ReceptacleController) DeleteById(ctx *gin.Context) {
	res := conf.Db.Where("id = ?", ctx.Param("id")).Delete(&model.ReceptacleTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *ReceptacleController) BatchDelete(ctx *gin.Context) {
	res := conf.Db.Where("pid = ?", ctx.Param("pid")).Delete(&model.ReceptacleTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ ImeiController（/imei） ============

type ImeiController struct{}

func (c *ImeiController) Binding(ctx *gin.Context) {
	res := conf.Db.Create(&model.ImeiTab{ChargingStation: ctx.Query("station"), ChargingSocket: ctx.Query("socket"), Status: 1})
	ctx.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
}

func (c *ImeiController) CancelBinding(ctx *gin.Context) {
	res := conf.Db.Where("charging_station = ? AND charging_socket = ?", ctx.Query("station"), ctx.Query("socket")).Delete(&model.ImeiTab{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *ImeiController) ListBinding(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.ImeiTab{}).Count(&total)
	var rows []model.ImeiTab
	conf.Db.Model(&model.ImeiTab{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]imeiDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, imeiToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (c *ImeiController) UpdateBinding(ctx *gin.Context) {
	id := ctx.Query("id")
	res := conf.Db.Model(&model.ImeiTab{}).Where("id = ?", id).Updates(map[string]interface{}{
		"charging_station": ctx.Query("station"),
		"charging_socket":  ctx.Query("socket"),
	})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}
