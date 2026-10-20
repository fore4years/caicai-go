package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/repository"
)

// ============================================================================
// ChargeStationController（二轮车充电站，base /ChargeStation）
// ReOrderPowerController（二轮车订单充电时段电量，base /receptacle/order/power）
// 严格对齐 Java 端 ChargeStationController / ReOrderPowerController 及其
// ServiceImpl + Mapper XML。经纬度直接用 longitude/latitude 两个字段。
// ============================================================================

// ChargeStationController 对齐 Java controller.ChargeStationController。
type ChargeStationController struct{}

// GetChargeStation GET /ChargeStation/getChargeStation?pid= 对齐 Java getChargeStation（getStationByPid）。
func (c *ChargeStationController) GetChargeStation(ctx *gin.Context) {
	s, err := repository.ChargingStationByPid(ctx.Query("pid"))
	if err != nil || s.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargeStationToDTO(*s)))
}

// GetById GET /ChargeStation/getById?id= 对齐 Java getById。
func (c *ChargeStationController) GetById(ctx *gin.Context) {
	s, err := repository.ChargingStationByID(ctx.Query("id"))
	if err != nil || s.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargeStationToDTO(*s)))
}

// GetByOpenid GET /ChargeStation/getByOpenid 对齐 Java getByOpenid（按当前用户 openid）。
func (c *ChargeStationController) GetByOpenid(ctx *gin.Context) {
	openid := ctx.GetString("openid")
	rows := repository.ChargingStationsByOpenid(openid)
	list := make([]chargeStationDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargeStationToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// Add POST /ChargeStation/add 对齐 Java add（@RequestBody ChargeStationDto）。
func (c *ChargeStationController) Add(ctx *gin.Context) {
	var req struct {
		Place     string `json:"place"`
		Name      string `json:"name"`
		Pid       string `json:"pid"`
		PowerMax  string `json:"powerMax"`
		Latitude  string `json:"latitude"`
		Longitude string `json:"longitude"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if req.Pid == "" {
		ctx.JSON(http.StatusOK, ResultError(400, "产品id不能为空!"))
		return
	}
	// 对齐 productService.getByProductId(pid)：产品 id 已存在则不允许重复添加。
	openid := ctx.GetString("openid")
	station, err := repository.ChargingStationByPidDB(req.Pid)
	exists := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}

	if exists {
		// ---------- 更新分支 ----------
		// BeanUtils.copyProperties(dto, station)：只覆盖 DTO 非空字段
		if req.Place != "" {
			station.Place = req.Place
		}
		if req.Name != "" {
			station.Name = req.Name
		}
		// powerMax：DTO 未传则保留原值
		if req.PowerMax != "" {
			if v, err := strconv.Atoi(req.PowerMax); err == nil {
				station.PowerMax = int32(v)
			}
		}
		// longitude/latitude 任一非空 → 更新经纬度
		if req.Longitude != "" || req.Latitude != "" {
			lng, _ := strconv.ParseFloat(req.Longitude, 64)
			lat, _ := strconv.ParseFloat(req.Latitude, 64)
			station.Point = model.NewPoint(lng, lat)
		}
		// pid 保持不变（Java 里 setPid(pid) 是显式保护）
		station.Openid = openid

		if err := repository.SaveChargingStation(station); err != nil {
			ctx.JSON(http.StatusOK, ResultSuccess(false))
			return
		}
		ctx.JSON(http.StatusOK, ResultSuccess(true))
		return
	}
	// 新增
	newStation := model.ChargingStationTbl{
		Place:    req.Place,
		Name:     req.Name,
		Pid:      req.Pid,
		Openid:   openid,
		PowerMax: 3000, // Java: setPowerMax("3000") 默认值
	}
	if req.PowerMax != "" {
		if v, err := strconv.Atoi(req.PowerMax); err == nil {
			newStation.PowerMax = int32(v)
		}
	}
	if req.Longitude != "" || req.Latitude != "" {
		lng, _ := strconv.ParseFloat(req.Longitude, 64)
		lat, _ := strconv.ParseFloat(req.Latitude, 64)
		newStation.Point = model.NewPoint(lng, lat)
	}

	if err := repository.SaveChargingStation(&newStation); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// GetByPoint GET /ChargeStation/getByPoint?longitude=&latitude=&current=&size= 对齐 Java getByPoint。
func (c *ChargeStationController) GetByPoint(ctx *gin.Context) {
	lngStr, latStr := ctx.Query("longitude"), ctx.Query("latitude")
	if lngStr == "" || latStr == "" {
		ctx.JSON(http.StatusOK, ResultError(400, "经纬度不能为空"))
		return
	}
	lng, _ := strconv.ParseFloat(lngStr, 64)
	lat, _ := strconv.ParseFloat(latStr, 64)
	current, size := parsePage(ctx)

	total := repository.CountChargingStations()
	rows := repository.ChargingStationsByPoint(lat, lng, size, pageOffset(current, size))

	list := make([]chargeStationDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargeStationToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetStationList GET /ChargeStation/getStationList 对齐 Java getStationList（list()）。
func (c *ChargeStationController) GetStationList(ctx *gin.Context) {
	rows, err := repository.AllChargingStations()
	if err != nil {
		logger.Mylog.Err(err).Msg("查询失败")
	}
	list := make([]chargeStationDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargeStationToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// ============================================================================
// ReOrderPowerController（/receptacle/order/power）
// ============================================================================

// ReOrderPowerController 对齐 Java controller.ReOrderPowerController。
type ReOrderPowerController struct{}

// reOrderPowerDTO 对应 Java domain.ReOrderPower（re_order_electronic_tbl，驼峰）。
type reOrderPowerDTO struct {
	CreateTime  *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime  *LocalDateTime `json:"updateTime,omitempty"`
	OrderID     string         `json:"orderId,omitempty"`
	Power       string         `json:"power,omitempty"`
	StartValue  float64        `json:"startValue,omitempty"`
	OverValue   float64        `json:"overValue,omitempty"`
	ID          int32          `json:"id"`
	PriceTimeID int32          `json:"priceTimeId,omitempty"`
	Count       int32          `json:"count,omitempty"`
}

func reOrderPowerToDTO(r model.ReOrderElectronicTbl) reOrderPowerDTO {
	return reOrderPowerDTO{
		ID:          r.ID,
		OrderID:     r.OrderID,
		StartValue:  r.StartValue.InexactFloat64(),
		OverValue:   r.OverValue.InexactFloat64(),
		PriceTimeID: r.PriceTimeID,
		Power:       r.Power,
		Count:       r.Count,
		CreateTime:  timeToLocal(r.CreateTime),
		UpdateTime:  timeToLocal(r.UpdateTime),
	}
}

// Add GET /receptacle/order/power/add?orderId=&meterValue= 对齐 Java add。
func (c *ReOrderPowerController) Add(ctx *gin.Context) {
	orderID := ctx.Query("orderId")
	var meterValue interface{}
	if s := ctx.Query("meterValue"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			meterValue = v
		}
	}
	err := repository.InsertReOrderPower(orderID, meterValue)
	ctx.JSON(http.StatusOK, ResultSuccess(err == nil))
}

// GetById GET /receptacle/order/power/getById?id= 对齐 Java getById。
func (c *ReOrderPowerController) GetById(ctx *gin.Context) {
	var r model.ReOrderElectronicTbl
	if err := repository.FindByID(&r, ctx.Query("id")).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(reOrderPowerToDTO(r)))
}

// GetAll GET /receptacle/order/power/getAll?current=&size= 对齐 Java getAll（page）。
func (c *ReOrderPowerController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.ReOrderElectronicTbl{}, &total)
	var rows []model.ReOrderElectronicTbl
	repository.Page(&model.ReOrderElectronicTbl{}, "", pageOffset(current, size), size, &rows)
	list := make([]reOrderPowerDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, reOrderPowerToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetValueByOrderid GET /receptacle/order/power/getValueByOrderid?orderid= 对齐 Java getValueByOrderid。
func (c *ReOrderPowerController) GetValueByOrderid(ctx *gin.Context) {
	d := repository.ReOrderPowerValueByOrderid(ctx.Query("orderid"))
	if d == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(d.Round(2)))
}

// GetFeesByOrderid GET /receptacle/order/power/getFeesByOrderid?orderid= 对齐 Java getFeesByOrderid。
func (c *ReOrderPowerController) GetFeesByOrderid(ctx *gin.Context) {
	d := repository.ReOrderPowerFeesByOrderid(ctx.Query("orderid"))
	if d == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(d.Round(2)))
}

// GetElectronicByOrderid GET /receptacle/order/power/getElectronicByOrderid?orderid=
// 对齐 Java getElectronicByOrderid（返回 {totalFee, serviceFee}）。
func (c *ReOrderPowerController) GetElectronicByOrderid(ctx *gin.Context) {
	totalFee, serviceFee := repository.ReOrderPowerElectronicByOrderid(ctx.Query("orderid"))
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{
		"totalFee":   totalFee.Round(2),
		"serviceFee": serviceFee.Round(2),
	}))
}

func (c *ReOrderPowerController) GetMaxPowerByOrderid(context *gin.Context) {
	maxPower := repository.ReOrderPowerMaxByOrderid(context.Query("orderid"))
	context.JSON(http.StatusOK, ResultSuccess(gin.H{
		"maxPower": maxPower.Round(2),
	}))
}
