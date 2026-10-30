package handler

import (
	"caicai-go/logger"
	"net/http"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ DTO（驼峰，对齐 Java Bean 序列化） ============

// chargingGunDTO 对应 Java model.ChargingGunBean（tab_charging_gun）。
type chargingGunDTO struct {
	UpdateTime *LocalDateTime  `json:"updateTime,omitempty"`
	Product    *productDTO     `json:"product,omitempty"`
	Name       string          `json:"name,omitempty"`
	UnitPrice  decimal.Decimal `json:"unitPrice,omitempty"`
	State      string          `json:"state,omitempty"`
	ProductID  string          `json:"productId,omitempty"`
	Direction  string          `json:"direction,omitempty"`
	ID         int32           `json:"id"`
	MeterValue int32           `json:"meterValue,omitempty"`
	Model      int32           `json:"model,omitempty"`
	Pwm        int32           `json:"pwm,omitempty"`
}

func chargingGunToDTO(g model.TabChargingGun) chargingGunDTO {
	dto := chargingGunDTO{
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
	// 关联产品（对齐 Java ChargingGunMapper.getById 的 <collection property="product">）
	if p := repository.ProductByPid(g.ProductID); p != nil {
		pd := productToDTO(*p)
		dto.Product = &pd
	}
	return dto
}

// stationManagementDTO 对应 Java domain.ChargingStationManagement。
type stationManagementDTO struct {
	CreateTime *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime *LocalDateTime `json:"updateTime,omitempty"`
	Openid     string         `json:"openid,omitempty"`
	Name       string         `json:"name,omitempty"`
	Place      string         `json:"place,omitempty"`
	ProductID  string         `json:"productId,omitempty"`
	Phone      string         `json:"phone,omitempty"`
	Email      string         `json:"email,omitempty"`
	Reviser    string         `json:"reviser,omitempty"`
	ID         int32          `json:"id"`
	ParkingNum int32          `json:"parkingNum,omitempty"`
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
	Status     int32          `json:"status"`
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
	CreatedTime     *LocalDateTime `json:"createdTime,omitempty"`
	UpdatedTime     *LocalDateTime `json:"updatedTime,omitempty"`
	ChargingStation string         `json:"chargingStation,omitempty"`
	ChargingSocket  string         `json:"chargingSocket,omitempty"`
	ID              int64          `json:"id"`
	Status          int32          `json:"status,omitempty"`
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
	if err := repository.FindByID(&g, ctx.Query("id")).Error; err != nil {
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
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&g).Error == nil))
}

func (c *GatewayController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.TabGateway{}, &total)
	var rows []model.TabGateway
	repository.Page(&model.TabGateway{}, "", pageOffset(current, size), size, &rows)
	list := make([]gatewayDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, gatewayToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// ============ LockController（/lock） ============

type LockController struct{}

func (c *LockController) GetById(ctx *gin.Context) {
	l, err := repository.LockByLockid(ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(lockToDTO(*l)))
}

func (c *LockController) GetByLockId(ctx *gin.Context) {
	l, err := repository.LockByLockid(strings.ToUpper(ctx.Query("lockId")))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(lockToDTO(*l)))
}

func (c *LockController) Add(ctx *gin.Context) {
	var l model.LockTbl
	if err := ctx.ShouldBindJSON(&l); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&l).Error == nil))
}

func (c *LockController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.LockTbl{}, &total)
	var rows []model.LockTbl
	repository.Page(&model.LockTbl{}, "", pageOffset(current, size), size, &rows)
	list := make([]lockDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, lockToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (c *LockController) GetByPid(ctx *gin.Context) {
	l, err := repository.LockByProductID(strings.ToUpper(ctx.Query("pid")))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(lockToDTO(*l)))
}

func (c *LockController) GetByOrder(ctx *gin.Context) {
	l, _, _, err := repository.LockProductGatewayByOrder(ctx.Query("orderid"))
	if err != nil || l.Lockid == "" {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(lockToDTO(l)))
}

// flexInt 兼容 JSON 数字与字符串两种形式的整数。
// 前端 chargingGun 的 model 以字符串 "220"/"380" 传输，Java Jackson 会自动转 Integer，这里对齐该行为。
type flexInt int32

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return err
	}
	*f = flexInt(n)
	return nil
}

// chargingGunAddReq 对应 Java ChargingGunBean 的请求字段（前端使用驼峰 productId，model 为字符串）。
type chargingGunAddReq struct {
	Direction string  `json:"direction"`
	ProductID string  `json:"productId"`
	Model     flexInt `json:"model"`
}

func (r chargingGunAddReq) toModel() model.TabChargingGun {
	return model.TabChargingGun{
		Direction: r.Direction,
		ProductID: r.ProductID,
		Model:     int32(r.Model),
	}
}

// ============ ChargingGunController（/chargingGun） ============

type ChargingGunController struct{}

func (c *ChargingGunController) GetById(ctx *gin.Context) {
	var g model.TabChargingGun
	if err := repository.FindByID(&g, ctx.Query("id")).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargingGunToDTO(g)))
}

func (c *ChargingGunController) Add(ctx *gin.Context) {
	var req chargingGunAddReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	g := req.toModel()
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&g).Error == nil))
}

func (c *ChargingGunController) AddPrivateGun(ctx *gin.Context) {
	var req chargingGunAddReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	g := req.toModel()
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&g).Error == nil))
}

func (c *ChargingGunController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.TabChargingGun{}, &total)
	var rows []model.TabChargingGun
	repository.Page(&model.TabChargingGun{}, "", pageOffset(current, size), size, &rows)
	list := make([]chargingGunDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargingGunToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (c *ChargingGunController) GetByPidAndDirection(ctx *gin.Context) {
	g := repository.ChargingGunByPidAndDirection(ctx.Query("pid"), ctx.Query("direction"))
	if g == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargingGunToDTO(g.Gun)))
}

func (c *ChargingGunController) UpdatePwm(ctx *gin.Context) {
	pwm, _ := strconv.Atoi(ctx.Query("pwm"))
	rowsAffected := repository.UpdateChargingGunPwm(ctx.Query("product_id"), ctx.Query("direction"), pwm)
	ctx.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}

// ============ ChargingStateController（/chargingState） ============

type ChargingStateController struct{}

func (c *ChargingStateController) List(ctx *gin.Context) {
	var rows []model.ChargingStateTbl
	repository.FindAll(&rows)
	list := make([]chargingStateDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargingStateToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *ChargingStateController) Page(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.ChargingStateTbl{}, &total)
	var rows []model.ChargingStateTbl
	repository.Page(&model.ChargingStateTbl{}, "create_time desc", pageOffset(current, size), size, &rows)
	list := make([]chargingStateDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargingStateToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (c *ChargingStateController) GetById(ctx *gin.Context) {
	var s model.ChargingStateTbl
	if err := repository.FindByID(&s, ctx.Query("id")).Error; err != nil {
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
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&s).Error == nil))
}

func (c *ChargingStateController) Update(ctx *gin.Context) {
	var s model.ChargingStateTbl
	if err := ctx.ShouldBindJSON(&s); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := repository.UpdateByID(&model.ChargingStateTbl{}, s.ID, s)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *ChargingStateController) DeleteById(ctx *gin.Context) {
	res := repository.DeleteByID(&model.ChargingStateTbl{}, ctx.Query("id"))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ StationManagementController（/chargingStations/management） ============

type StationManagementController struct{}

func (c *StationManagementController) GetAll(ctx *gin.Context) {
	var rows []model.ChargingStationManagementTbl
	repository.FindAll(&rows)
	list := make([]stationManagementDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, stationManagementToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *StationManagementController) GetById(ctx *gin.Context) {
	var m model.ChargingStationManagementTbl
	if err := repository.FindByID(&m, ctx.Param("id")).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(stationManagementToDTO(m)))
}

func (c *StationManagementController) GetByOpenId(ctx *gin.Context) {
	rows := repository.StationManagementsByOpenid(ctx.Param("openid"))
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
	if m.ProductID == "" {
		ctx.JSON(http.StatusOK, ResultError(400, "产品id不能为空"))
		return
	}
	if repository.StationManagementExistsByProductID(m.ProductID) {
		ctx.JSON(http.StatusOK, ResultError(400, "该充电站id已存在,请检查后重试!"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&m).Error == nil))
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
	res := repository.UpdateByID(&model.ChargingStationManagementTbl{}, m.ID, m)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *StationManagementController) Delete(ctx *gin.Context) {
	res := repository.DeleteByID(&model.ChargingStationManagementTbl{}, ctx.Param("id"))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ LedController（/led） ============

type LedController struct{}

func (c *LedController) GetById(ctx *gin.Context) {
	var l model.LedTbl
	if err := repository.FindByID(&l, ctx.Param("id")).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(ledToDTO(l)))
}

// ============ ReceptacleController（/receptacle） ============

type ReceptacleController struct{}

func (c *ReceptacleController) GetReceptacleAll(ctx *gin.Context) {
	rows := repository.ReceptaclesByPid(ctx.Query("pid"))
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
	ctx.JSON(http.StatusOK, ResultSuccess(repository.Create(&r).Error == nil))
}

func (c *ReceptacleController) BatchAdd(ctx *gin.Context) {
	// body {pid, num}，对齐 Java ReceptacleDto。
	var req struct {
		Pid string `json:"pid"`
		Num string `json:"num"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	num, _ := strconv.Atoi(req.Num)
	if num <= 0 {
		ctx.JSON(http.StatusOK, ResultError(400, "数量必须为正数"))
		return
	}
	rows := make([]model.ReceptacleTbl, 0, num)
	for i := 1; i <= num; i++ {
		rows = append(rows, model.ReceptacleTbl{Pid: req.Pid, Number: int32(i), Status: 0, Name: strconv.Itoa(i) + "号插座"})
	}
	tx := repository.Create(&rows)
	if tx.Error != nil {
		logger.Mylog.Err(tx.Error).Msg("添加插座失败")
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

func (c *ReceptacleController) DeleteById(ctx *gin.Context) {
	res := repository.DeleteByID(&model.ReceptacleTbl{}, ctx.Param("id"))
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *ReceptacleController) BatchDelete(ctx *gin.Context) {
	repository.DeleteReceptaclesByPid(ctx.Param("pid"))
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ ImeiController（/imei） ============

type ImeiController struct{}

func (c *ImeiController) Binding(ctx *gin.Context) {
	station := ctx.Query("station")
	socket := ctx.Query("socket")

	// 已存在绑定关系直接返回成功（对齐 Java ImeiServiceImpl.binding 的 checkBinding）。
	if repository.ImeiBindingExists(station, socket) {
		ctx.JSON(http.StatusOK, ResultSuccess(true))
		return
	}

	res := repository.Create(&model.ImeiTab{ChargingStation: station, ChargingSocket: socket, Status: 1})
	ctx.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
}

func (c *ImeiController) CancelBinding(ctx *gin.Context) {
	station := ctx.Query("station")
	socket := ctx.Query("socket")

	// 不存在绑定关系直接返回成功（对齐 Java ImeiServiceImpl.cancelBinding）。
	if !repository.ImeiBindingExists(station, socket) {
		ctx.JSON(http.StatusOK, ResultSuccess(true))
		return
	}

	// 逻辑删除（@TableLogic -> deleted = 1），对齐 Java remove()。
	rowsAffected := repository.DeleteImeiBinding(station, socket)
	ctx.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}

func (c *ImeiController) ListBinding(ctx *gin.Context) {
	current, size := parsePage(ctx)
	total, rows := repository.ListImeiBindings(pageOffset(current, size), size)
	list := make([]imeiDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, imeiToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

func (c *ImeiController) UpdateBinding(ctx *gin.Context) {
	id := ctx.Query("id")
	res := repository.UpdateByID(&model.ImeiTab{}, id, map[string]interface{}{
		"charging_station": ctx.Query("station"),
		"charging_socket":  ctx.Query("socket"),
	})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}
