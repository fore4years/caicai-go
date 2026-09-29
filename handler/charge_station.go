package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// ChargeStationController（二轮车充电站，base /ChargeStation）
// ReOrderPowerController（二轮车订单充电时段电量，base /receptacle/order/power）
// 严格对齐 Java 端 ChargeStationController / ReOrderPowerController 及其
// ServiceImpl + Mapper XML。point 用 ST_AsText 输出 WKT（对齐 Java JTS Point.toText()）。
// ============================================================================

// chargeStationGeoCols 充电站地理查询列（point 输出 WKT）。
const chargeStationGeoCols = "id, ST_AsText(point) AS point, place, name, openid, power_max, pid, create_time, update_time"

// ChargeStationController 对齐 Java controller.ChargeStationController。
type ChargeStationController struct{}

// GetChargeStation GET /ChargeStation/getChargeStation?pid= 对齐 Java getChargeStation（getStationByPid）。
func (c *ChargeStationController) GetChargeStation(ctx *gin.Context) {
	var s model.ChargingStationTbl
	if err := conf.Db.Raw("SELECT "+chargeStationGeoCols+" FROM charging_station_tbl WHERE pid = ?", ctx.Query("pid")).Scan(&s).Error; err != nil || s.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargeStationToDTO(s)))
}

// GetById GET /ChargeStation/getById?id= 对齐 Java getById。
func (c *ChargeStationController) GetById(ctx *gin.Context) {
	var s model.ChargingStationTbl
	if err := conf.Db.Raw("SELECT "+chargeStationGeoCols+" FROM charging_station_tbl WHERE id = ?", ctx.Query("id")).Scan(&s).Error; err != nil || s.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargeStationToDTO(s)))
}

// GetByOpenid GET /ChargeStation/getByOpenid 对齐 Java getByOpenid（按当前用户 openid）。
func (c *ChargeStationController) GetByOpenid(ctx *gin.Context) {
	openid := ctx.GetString("openid")
	var rows []model.ChargingStationTbl
	conf.Db.Raw("SELECT "+chargeStationGeoCols+" FROM charging_station_tbl WHERE openid = ?", openid).Scan(&rows)
	list := make([]chargeStationDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargeStationToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// Add POST /ChargeStation/add 对齐 Java add（@RequestBody ChargeStationDto）。
func (c *ChargeStationController) Add(ctx *gin.Context) {
	var req struct {
		Place     string   `json:"place"`
		Name      string   `json:"name"`
		Pid       string   `json:"pid"`
		PowerMax  string   `json:"powerMax"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
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
	var n int64
	conf.Db.Model(&model.TabProduct{}).Where("pid = ?", req.Pid).Count(&n)
	if n > 0 {
		ctx.JSON(http.StatusOK, ResultError(400, "此产品id已存在,请勿重复添加!"))
		return
	}

	openid := ctx.GetString("openid")
	powerMax, _ := strconv.Atoi(req.PowerMax)

	// 对齐 Java：longitude/latitude 任一非空才构造 point，否则 point 为 NULL。
	var wkt interface{}
	if req.Longitude != nil || req.Latitude != nil {
		var lng, lat float64
		if req.Longitude != nil {
			lng = *req.Longitude
		}
		if req.Latitude != nil {
			lat = *req.Latitude
		}
		wkt = wktPoint(lng, lat)
	}

	if err := conf.Db.Exec(
		"INSERT INTO charging_station_tbl (point, place, name, openid, pid, power_max) VALUES (ST_GeomFromText(?),?,?,?,?,?)",
		wkt, req.Place, req.Name, openid, req.Pid, powerMax,
	).Error; err != nil {
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
	wkt := wktPoint(lng, lat)
	current, size := parsePage(ctx)

	var total int64
	conf.Db.Model(&model.ChargingStationTbl{}).Count(&total)

	var rows []model.ChargingStationTbl
	conf.Db.Raw(
		"SELECT "+chargeStationGeoCols+", ST_Distance_Sphere(point, ST_GeomFromText(?)) AS distance FROM charging_station_tbl "+
			"ORDER BY ST_Distance_Sphere(point, ST_GeomFromText(?)) ASC LIMIT ? OFFSET ?",
		wkt, wkt, size, pageOffset(current, size),
	).Scan(&rows)

	list := make([]chargeStationDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, chargeStationToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetStationList GET /ChargeStation/getStationList 对齐 Java getStationList（list()）。
func (c *ChargeStationController) GetStationList(ctx *gin.Context) {
	var rows []model.ChargingStationTbl
	conf.Db.Raw("SELECT " + chargeStationGeoCols + " FROM charging_station_tbl").Scan(&rows)
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
	ID          int32          `json:"id"`
	OrderID     string         `json:"orderId,omitempty"`
	StartValue  float64        `json:"startValue,omitempty"`
	OverValue   float64        `json:"overValue,omitempty"`
	PriceTimeID int32          `json:"priceTimeId,omitempty"`
	Power       string         `json:"power,omitempty"`
	Count       int32          `json:"count,omitempty"`
	CreateTime  *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime  *LocalDateTime `json:"updateTime,omitempty"`
}

func reOrderPowerToDTO(r model.ReOrderElectronicTbl) reOrderPowerDTO {
	return reOrderPowerDTO{
		ID:          r.ID,
		OrderID:     r.OrderID,
		StartValue:  r.StartValue,
		OverValue:   r.OverValue,
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
	err := conf.Db.Exec(
		"INSERT INTO re_order_electronic_tbl (order_id, start_value, over_value, price_time_id) "+
			"VALUES (?,?,?,(SELECT id FROM tab_price_time WHERE start_time < curtime() AND over_time >= curtime()))",
		orderID, meterValue, meterValue,
	).Error
	ctx.JSON(http.StatusOK, ResultSuccess(err == nil))
}

// GetById GET /receptacle/order/power/getById?id= 对齐 Java getById。
func (c *ReOrderPowerController) GetById(ctx *gin.Context) {
	var r model.ReOrderElectronicTbl
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&r).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(reOrderPowerToDTO(r)))
}

// GetAll GET /receptacle/order/power/getAll?current=&size= 对齐 Java getAll（page）。
func (c *ReOrderPowerController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.ReOrderElectronicTbl{}).Count(&total)
	var rows []model.ReOrderElectronicTbl
	conf.Db.Model(&model.ReOrderElectronicTbl{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]reOrderPowerDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, reOrderPowerToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetValueByOrderid GET /receptacle/order/power/getValueByOrderid?orderid= 对齐 Java getValueByOrderid。
func (c *ReOrderPowerController) GetValueByOrderid(ctx *gin.Context) {
	d := queryFeeDecimal("SELECT (MAX(over_value)-MIN(start_value)) FROM re_order_electronic_tbl WHERE order_id = ?", ctx.Query("orderid"))
	if d == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(d.Round(2)))
}

// GetFeesByOrderid GET /receptacle/order/power/getFeesByOrderid?orderid= 对齐 Java getFeesByOrderid。
func (c *ReOrderPowerController) GetFeesByOrderid(ctx *gin.Context) {
	d := queryFeeDecimal(`SELECT SUM((over_value - start_value) * (charging_price_tbl.total + charging_price_tbl.service))
		FROM re_order_electronic_tbl
		LEFT JOIN tab_price_time ON tab_price_time.id = re_order_electronic_tbl.price_time_id
		LEFT JOIN charging_price_tbl ON charging_price_tbl.id = tab_price_time.price_id
		WHERE re_order_electronic_tbl.order_id = ?`, ctx.Query("orderid"))
	if d == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(d.Round(2)))
}

// GetElectronicByOrderid GET /receptacle/order/power/getElectronicByOrderid?orderid=
// 对齐 Java getElectronicByOrderid（返回 {totalFee, serviceFee}）。
func (c *ReOrderPowerController) GetElectronicByOrderid(ctx *gin.Context) {
	var totalFee, serviceFee decimal.Decimal
	if err := conf.Db.Raw(`SELECT
			SUM((over_value - start_value) * charging_price_tbl.total) AS totalFee,
			SUM((over_value - start_value) * charging_price_tbl.service) AS serviceFee
		FROM re_order_electronic_tbl
		LEFT JOIN tab_price_time ON tab_price_time.id = re_order_electronic_tbl.price_time_id
		LEFT JOIN charging_price_tbl ON charging_price_tbl.id = tab_price_time.price_id
		WHERE re_order_electronic_tbl.order_id = ?`, ctx.Query("orderid")).Row().Scan(&totalFee, &serviceFee); err != nil {
		totalFee, serviceFee = decimal.Zero, decimal.Zero
	}
	ctx.JSON(http.StatusOK, ResultSuccess(gin.H{
		"totalFee":   totalFee.Round(2),
		"serviceFee": serviceFee.Round(2),
	}))
}

// RegisterChargeStation 注册 ChargeStationController 与 ReOrderPowerController 全部路由。
func RegisterChargeStation(r *gin.Engine) {
	chargeStation := new(ChargeStationController)
	reOrderPower := new(ReOrderPowerController)

	cs := r.Group("/ChargeStation")
	{
		cs.GET("/getChargeStation", chargeStation.GetChargeStation)
		cs.GET("/getById", chargeStation.GetById)
		cs.GET("/getByOpenid", chargeStation.GetByOpenid)
		cs.POST("/add", chargeStation.Add)
		cs.GET("/getByPoint", chargeStation.GetByPoint)
		cs.GET("/getStationList", chargeStation.GetStationList)
	}

	rop := r.Group("/receptacle/order/power")
	{
		rop.GET("/add", reOrderPower.Add)
		rop.GET("/getById", reOrderPower.GetById)
		rop.GET("/getAll", reOrderPower.GetAll)
		rop.GET("/getValueByOrderid", reOrderPower.GetValueByOrderid)
		rop.GET("/getFeesByOrderid", reOrderPower.GetFeesByOrderid)
		rop.GET("/getElectronicByOrderid", reOrderPower.GetElectronicByOrderid)
	}
}
