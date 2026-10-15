package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/model"
	"caicai-go/repository"
)

// priceTimeDTO 对应 Java ChargingPriceTimeBean / PriceTimeBean。
// startTime/overTime 对齐 Java LocalTime 的 ISO 序列化（HH:mm:ss）。
type priceTimeDTO struct {
	Price     *repository.PriceDTO `json:"price,omitempty"`
	StartTime string               `json:"startTime,omitempty"`
	OverTime  string               `json:"overTime,omitempty"`
	ID        int32                `json:"id,omitempty"`
	PriceID   int32                `json:"priceId,omitempty"`
}

// chargingPriceToDTO 将 model.ChargingPriceTbl 转为 repository.PriceDTO（只保留 Java bean 的四列）。
func chargingPriceToDTO(p model.ChargingPriceTbl) repository.PriceDTO {
	return repository.PriceDTO{ID: p.ID, Total: p.Total, Name: p.Name, Service: p.Service}
}

// tabPriceToDTO 将 model.TabPrice 转为 repository.PriceDTO（只保留 Java bean 的四列）。
func tabPriceToDTO(p model.TabPrice) repository.PriceDTO {
	return repository.PriceDTO{ID: p.ID, Total: p.Total, Name: p.Name, Service: p.Service}
}

// findChargingPrice 按 price_id 查询 charging_price_tbl（Java 的 collection select N+1）。
func findChargingPrice(id int32) *repository.PriceDTO {
	p, err := repository.ChargingPriceByID(id)
	if err != nil {
		return nil
	}
	v := chargingPriceToDTO(*p)
	return &v
}

// findTabPrice 按 price_id 查询 tab_price（Java 的 collection select N+1）。
func findTabPrice(id int32) *repository.PriceDTO {
	p, err := repository.TabPriceByID(id)
	if err != nil {
		return nil
	}
	v := tabPriceToDTO(*p)
	return &v
}

// ============ ChargingPriceController（/ChargingPrice，表 charging_price_tbl） ============

type ChargingPriceController struct{}

// GetById GET /ChargingPrice/getById 对齐 Java getById：按主键查 charging_price_tbl。
func (c *ChargingPriceController) GetById(ctx *gin.Context) {
	p, err := repository.ChargingPriceByID(ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargingPriceToDTO(*p)))
}

// Add POST /ChargingPrice/add 对齐 Java add：insert 一条 charging_price_tbl（仅 total/name/service）。
func (c *ChargingPriceController) Add(ctx *gin.Context) {
	var v model.ChargingPriceTbl
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	err := repository.InsertChargingPrice(&v)
	ctx.JSON(http.StatusOK, ResultSuccess(err == nil))
}

// GetAll GET /ChargingPrice/getAll 对齐 Java getAll：MyBatis-Plus 分页（无排序）。
func (c *ChargingPriceController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.ChargingPriceTbl{}, &total)
	var rows []model.ChargingPriceTbl
	repository.Page(&model.ChargingPriceTbl{}, "", pageOffset(current, size), size, &rows)
	list := make([]repository.PriceDTO, 0, len(rows))
	for _, p := range rows {
		list = append(list, chargingPriceToDTO(p))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetMix GET /ChargingPrice/getMix 对齐 Java getMix：按 total 升序取第一条（最低电价）。
func (c *ChargingPriceController) GetMix(ctx *gin.Context) {
	p, err := repository.LowestChargingPrice()
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargingPriceToDTO(*p)))
}

// GetByTime GET /ChargingPrice/getByTime 对齐 Java getMixByTime：
// tab_price_time LEFT JOIN charging_price_tbl，over_time>=start AND start_time<=end，
// 按 total+service 升序取第一条。
func (c *ChargingPriceController) GetByTime(ctx *gin.Context) {
	start := ctx.Query("start")
	end := ctx.Query("end")
	rows := repository.ChargingPriceByTime(start, end)
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(rows[0]))
}

// ============ ChargingPriceTimeController（/ChargingPriceTime，表 tab_price_time + charging_price_tbl） ============

type ChargingPriceTimeController struct{}

// GetById GET /ChargingPriceTime/getById 对齐 Java getById：查时段 + 嵌套电价。
func (c *ChargingPriceTimeController) GetById(ctx *gin.Context) {
	rows := repository.PriceTimeRowByID(ctx.Query("id"))
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	r := rows[0]
	dto := priceTimeDTO{ID: r.ID, StartTime: r.StartTime, OverTime: r.OverTime, PriceID: r.PriceID}
	dto.Price = findChargingPrice(r.PriceID)
	ctx.JSON(http.StatusOK, ResultSuccess(dto))
}

// Add POST /ChargingPriceTime/add 对齐 Java add：insert 一条 tab_price_time。
func (c *ChargingPriceTimeController) Add(ctx *gin.Context) {
	var req struct {
		StartTime string `json:"startTime"`
		OverTime  string `json:"overTime"`
		PriceID   int32  `json:"priceId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	err := repository.InsertTabPriceTime(req.StartTime, req.OverTime, req.PriceID)
	ctx.JSON(http.StatusOK, ResultSuccess(err == nil))
}

// GetAll GET /ChargingPriceTime/getAll 对齐 Java getAll：MyBatis-Plus 分页（无排序）。
func (c *ChargingPriceTimeController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.TabPriceTime{}, &total)
	rows := repository.PriceTimeRowsAll(pageOffset(current, size), size)
	list := make([]priceTimeDTO, 0, len(rows))
	for _, r := range rows {
		dto := priceTimeDTO{ID: r.ID, StartTime: r.StartTime, OverTime: r.OverTime, PriceID: r.PriceID}
		dto.Price = findChargingPrice(r.PriceID)
		list = append(list, dto)
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetByTime GET /ChargingPriceTime/getByTime 对齐 Java getMixByTime：
// tab_price_time LEFT JOIN charging_price_tbl，over_time>=start AND start_time<=end，
// 按 total+service 升序取第一条（含嵌套电价）。
func (c *ChargingPriceTimeController) GetByTime(ctx *gin.Context) {
	start := ctx.Query("start")
	end := ctx.Query("end")
	rows := repository.ChargingPriceTimeByTime(start, end)
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	r := rows[0]
	dto := priceTimeDTO{ID: r.ID, StartTime: r.StartTime, OverTime: r.OverTime, PriceID: r.PriceID}
	dto.Price = findChargingPrice(r.PriceID)
	ctx.JSON(http.StatusOK, ResultSuccess(dto))
}

// GetNow GET /ChargingPriceTime/getNow 对齐 Java getNow：
// start_time<now AND over_time>=now，按 over_time 降序取第一条的 price.total。
func (c *ChargingPriceTimeController) GetNow(ctx *gin.Context) {
	totals := repository.ChargingPriceTotalNow()
	if len(totals) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(totals[0]))
}

// ============ PriceController（/price，表 tab_price） ============

type PriceController struct{}

// GetById GET /price/getById 对齐 Java getById：按主键查 tab_price。
func (c *PriceController) GetById(ctx *gin.Context) {
	p, err := repository.TabPriceByID(ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(tabPriceToDTO(*p)))
}

// Add POST /price/add 对齐 Java add：insert 一条 tab_price（仅 total/name/service）。
func (c *PriceController) Add(ctx *gin.Context) {
	var v model.TabPrice
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	err := repository.InsertTabPrice(&v)
	ctx.JSON(http.StatusOK, ResultSuccess(err == nil))
}

// GetAll GET /price/getAll 对齐 Java getAll：MyBatis-Plus 分页（无排序）。
func (c *PriceController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.TabPrice{}, &total)
	var rows []model.TabPrice
	repository.Page(&model.TabPrice{}, "", pageOffset(current, size), size, &rows)
	list := make([]repository.PriceDTO, 0, len(rows))
	for _, p := range rows {
		list = append(list, tabPriceToDTO(p))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetMix GET /price/getMix 对齐 Java getMix：按 total 升序取第一条（最低电价）。
func (c *PriceController) GetMix(ctx *gin.Context) {
	p, err := repository.LowestTabPrice()
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(tabPriceToDTO(*p)))
}

// GetByTime GET /price/getByTime 对齐 Java getMixByTime：
// tab_price_time LEFT JOIN tab_price，over_time>=start AND start_time<=end，
// 按 total+service 升序取第一条。
func (c *PriceController) GetByTime(ctx *gin.Context) {
	start := ctx.Query("start")
	end := ctx.Query("end")
	rows := repository.TabPriceByTime(start, end)
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(rows[0]))
}

// ============ PriceTimeController（/priceTime，表 tab_price_time + tab_price） ============

type PriceTimeController struct{}

// GetById GET /priceTime/getById 对齐 Java getById：查时段 + 嵌套电价。
func (c *PriceTimeController) GetById(ctx *gin.Context) {
	rows := repository.PriceTimeRowByID(ctx.Query("id"))
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	r := rows[0]
	dto := priceTimeDTO{ID: r.ID, StartTime: r.StartTime, OverTime: r.OverTime, PriceID: r.PriceID}
	dto.Price = findTabPrice(r.PriceID)
	ctx.JSON(http.StatusOK, ResultSuccess(dto))
}

// Add POST /priceTime/add 对齐 Java add：insert 一条 tab_price_time。
func (c *PriceTimeController) Add(ctx *gin.Context) {
	var req struct {
		StartTime string `json:"startTime"`
		OverTime  string `json:"overTime"`
		PriceID   int32  `json:"priceId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	err := repository.InsertTabPriceTime(req.StartTime, req.OverTime, req.PriceID)
	ctx.JSON(http.StatusOK, ResultSuccess(err == nil))
}

// GetAll GET /priceTime/getAll 对齐 Java getAll：MyBatis-Plus 分页（无排序）。
func (c *PriceTimeController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.TabPriceTime{}, &total)
	rows := repository.PriceTimeRowsAll(pageOffset(current, size), size)
	list := make([]priceTimeDTO, 0, len(rows))
	for _, r := range rows {
		dto := priceTimeDTO{ID: r.ID, StartTime: r.StartTime, OverTime: r.OverTime, PriceID: r.PriceID}
		dto.Price = findTabPrice(r.PriceID)
		list = append(list, dto)
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetByTime GET /priceTime/getByTime 对齐 Java getMixByTime：
// tab_price_time LEFT JOIN tab_price，over_time>=start AND start_time<=end，
// 按 total+service 升序取第一条（含嵌套电价）。
func (c *PriceTimeController) GetByTime(ctx *gin.Context) {
	start := ctx.Query("start")
	end := ctx.Query("end")
	rows := repository.TabPriceTimeByTime(start, end)
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	r := rows[0]
	dto := priceTimeDTO{ID: r.ID, StartTime: r.StartTime, OverTime: r.OverTime, PriceID: r.PriceID}
	dto.Price = findTabPrice(r.PriceID)
	ctx.JSON(http.StatusOK, ResultSuccess(dto))
}

// GetNow GET /priceTime/getNow 对齐 Java getNow：
// 当前时段（start_time<now AND over_time>=now，over_time 降序第一条）的 price，
// 返回 total + service*0.8。
func (c *PriceTimeController) GetNow(ctx *gin.Context) {
	total, service, ok := repository.TabPriceNow()
	if !ok {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	result := total.Add(service.Mul(decimal.NewFromFloat(0.8)))
	ctx.JSON(http.StatusOK, ResultSuccess(result))
}

// GetAllNow GET /priceTime/getAllNow 对齐 Java getAllNow：
// 当前时段（同上）的第一条的嵌套 price（PriceBean）。
func (c *PriceTimeController) GetAllNow(ctx *gin.Context) {
	rows := repository.TabPriceAllNow()
	if len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(rows[0]))
}
