package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// priceDTO 对应 Java ChargingPriceBean / PriceBean（字段一致：id/total/name/service）。
type priceDTO struct {
	ID      int32           `json:"id,omitempty"`
	Total   decimal.Decimal `json:"total,omitempty"`
	Name    string          `json:"name,omitempty"`
	Service decimal.Decimal `json:"service,omitempty"`
}

// priceTimeDTO 对应 Java ChargingPriceTimeBean / PriceTimeBean。
// startTime/overTime 对齐 Java LocalTime 的 ISO 序列化（HH:mm:ss）。
type priceTimeDTO struct {
	ID        int32     `json:"id,omitempty"`
	StartTime string    `json:"startTime,omitempty"`
	OverTime  string    `json:"overTime,omitempty"`
	PriceID   int32     `json:"priceId,omitempty"`
	Price     *priceDTO `json:"price,omitempty"`
}

// chargingPriceToDTO 将 model.ChargingPriceTbl 转为 priceDTO（只保留 Java bean 的四列）。
func chargingPriceToDTO(p model.ChargingPriceTbl) priceDTO {
	return priceDTO{ID: p.ID, Total: p.Total, Name: p.Name, Service: p.Service}
}

// tabPriceToDTO 将 model.TabPrice 转为 priceDTO（只保留 Java bean 的四列）。
func tabPriceToDTO(p model.TabPrice) priceDTO {
	return priceDTO{ID: p.ID, Total: p.Total, Name: p.Name, Service: p.Service}
}

// findChargingPrice 按 price_id 查询 charging_price_tbl（Java 的 collection select N+1）。
func findChargingPrice(id int32) *priceDTO {
	var p model.ChargingPriceTbl
	if err := conf.Db.Where("id = ?", id).First(&p).Error; err != nil {
		return nil
	}
	v := chargingPriceToDTO(p)
	return &v
}

// findTabPrice 按 price_id 查询 tab_price（Java 的 collection select N+1）。
func findTabPrice(id int32) *priceDTO {
	var p model.TabPrice
	if err := conf.Db.Where("id = ?", id).First(&p).Error; err != nil {
		return nil
	}
	v := tabPriceToDTO(p)
	return &v
}

// priceTimeRow 用于扫描 tab_price_time 的行（时间列转 HH:mm:ss 字符串，避免 TIME 扫描歧义）。
type priceTimeRow struct {
	ID        int32
	StartTime string
	OverTime  string
	PriceID   int32
}

// ============ ChargingPriceController（/ChargingPrice，表 charging_price_tbl） ============

type ChargingPriceController struct{}

// GetById GET /ChargingPrice/getById 对齐 Java getById：按主键查 charging_price_tbl。
func (c *ChargingPriceController) GetById(ctx *gin.Context) {
	var p model.ChargingPriceTbl
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargingPriceToDTO(p)))
}

// Add POST /ChargingPrice/add 对齐 Java add：insert 一条 charging_price_tbl（仅 total/name/service）。
func (c *ChargingPriceController) Add(ctx *gin.Context) {
	var v model.ChargingPriceTbl
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Select("total", "name", "service").Create(&v)
	ctx.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
}

// GetAll GET /ChargingPrice/getAll 对齐 Java getAll：MyBatis-Plus 分页（无排序）。
func (c *ChargingPriceController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.ChargingPriceTbl{}).Count(&total)
	var rows []model.ChargingPriceTbl
	conf.Db.Model(&model.ChargingPriceTbl{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]priceDTO, 0, len(rows))
	for _, p := range rows {
		list = append(list, chargingPriceToDTO(p))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetMix GET /ChargingPrice/getMix 对齐 Java getMix：按 total 升序取第一条（最低电价）。
func (c *ChargingPriceController) GetMix(ctx *gin.Context) {
	var p model.ChargingPriceTbl
	if err := conf.Db.Order("total asc").First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargingPriceToDTO(p)))
}

// GetByTime GET /ChargingPrice/getByTime 对齐 Java getMixByTime：
// tab_price_time LEFT JOIN charging_price_tbl，over_time>=start AND start_time<=end，
// 按 total+service 升序取第一条。
func (c *ChargingPriceController) GetByTime(ctx *gin.Context) {
	start := ctx.Query("start")
	end := ctx.Query("end")
	var rows []priceDTO
	err := conf.Db.Raw(
		`SELECT charging_price_tbl.id, charging_price_tbl.total, charging_price_tbl.name, charging_price_tbl.service
		 FROM tab_price_time
		 LEFT JOIN charging_price_tbl ON tab_price_time.price_id = charging_price_tbl.id
		 WHERE tab_price_time.over_time >= ? AND tab_price_time.start_time <= ?
		 ORDER BY (charging_price_tbl.total + charging_price_tbl.service) LIMIT 1`,
		start, end,
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(rows[0]))
}

// ============ ChargingPriceTimeController（/ChargingPriceTime，表 tab_price_time + charging_price_tbl） ============

type ChargingPriceTimeController struct{}

// GetById GET /ChargingPriceTime/getById 对齐 Java getById：查时段 + 嵌套电价。
func (c *ChargingPriceTimeController) GetById(ctx *gin.Context) {
	var rows []priceTimeRow
	err := conf.Db.Raw(
		`SELECT id, TIME_FORMAT(start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(over_time,'%H:%i:%s') AS over_time, price_id
		 FROM tab_price_time WHERE id = ?`, ctx.Query("id"),
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
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
	res := conf.Db.Exec(
		"INSERT INTO tab_price_time (start_time, over_time, price_id) VALUES (?, ?, ?)",
		req.StartTime, req.OverTime, req.PriceID,
	)
	ctx.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
}

// GetAll GET /ChargingPriceTime/getAll 对齐 Java getAll：MyBatis-Plus 分页（无排序）。
func (c *ChargingPriceTimeController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.TabPriceTime{}).Count(&total)
	var rows []priceTimeRow
	conf.Db.Raw(
		`SELECT id, TIME_FORMAT(start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(over_time,'%H:%i:%s') AS over_time, price_id
		 FROM tab_price_time LIMIT ? OFFSET ?`,
		size, pageOffset(current, size),
	).Scan(&rows)
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
	var rows []priceTimeRow
	err := conf.Db.Raw(
		`SELECT tab_price_time.id, TIME_FORMAT(tab_price_time.start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(tab_price_time.over_time,'%H:%i:%s') AS over_time, tab_price_time.price_id
		 FROM tab_price_time
		 LEFT JOIN charging_price_tbl ON tab_price_time.price_id = charging_price_tbl.id
		 WHERE tab_price_time.over_time >= ? AND tab_price_time.start_time <= ?
		 ORDER BY (charging_price_tbl.total + charging_price_tbl.service) LIMIT 1`,
		start, end,
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
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
	var totals []decimal.Decimal
	err := conf.Db.Raw(
		`SELECT charging_price_tbl.total
		 FROM tab_price_time
		 LEFT JOIN charging_price_tbl ON tab_price_time.price_id = charging_price_tbl.id
		 WHERE tab_price_time.start_time < CURTIME() AND tab_price_time.over_time >= CURTIME()
		 ORDER BY tab_price_time.over_time DESC LIMIT 1`,
	).Scan(&totals).Error
	if err != nil || len(totals) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(totals[0]))
}

// ============ PriceController（/price，表 tab_price） ============

type PriceController struct{}

// GetById GET /price/getById 对齐 Java getById：按主键查 tab_price。
func (c *PriceController) GetById(ctx *gin.Context) {
	var p model.TabPrice
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(tabPriceToDTO(p)))
}

// Add POST /price/add 对齐 Java add：insert 一条 tab_price（仅 total/name/service）。
func (c *PriceController) Add(ctx *gin.Context) {
	var v model.TabPrice
	if err := ctx.ShouldBindJSON(&v); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Select("total", "name", "service").Create(&v)
	ctx.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
}

// GetAll GET /price/getAll 对齐 Java getAll：MyBatis-Plus 分页（无排序）。
func (c *PriceController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.TabPrice{}).Count(&total)
	var rows []model.TabPrice
	conf.Db.Model(&model.TabPrice{}).Offset(pageOffset(current, size)).Limit(size).Find(&rows)
	list := make([]priceDTO, 0, len(rows))
	for _, p := range rows {
		list = append(list, tabPriceToDTO(p))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetMix GET /price/getMix 对齐 Java getMix：按 total 升序取第一条（最低电价）。
func (c *PriceController) GetMix(ctx *gin.Context) {
	var p model.TabPrice
	if err := conf.Db.Order("total asc").First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(tabPriceToDTO(p)))
}

// GetByTime GET /price/getByTime 对齐 Java getMixByTime：
// tab_price_time LEFT JOIN tab_price，over_time>=start AND start_time<=end，
// 按 total+service 升序取第一条。
func (c *PriceController) GetByTime(ctx *gin.Context) {
	start := ctx.Query("start")
	end := ctx.Query("end")
	var rows []priceDTO
	err := conf.Db.Raw(
		`SELECT tab_price.id, tab_price.total, tab_price.name, tab_price.service
		 FROM tab_price_time
		 LEFT JOIN tab_price ON tab_price_time.price_id = tab_price.id
		 WHERE tab_price_time.over_time >= ? AND tab_price_time.start_time <= ?
		 ORDER BY (tab_price.total + tab_price.service) LIMIT 1`,
		start, end,
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(rows[0]))
}

// ============ PriceTimeController（/priceTime，表 tab_price_time + tab_price） ============

type PriceTimeController struct{}

// GetById GET /priceTime/getById 对齐 Java getById：查时段 + 嵌套电价。
func (c *PriceTimeController) GetById(ctx *gin.Context) {
	var rows []priceTimeRow
	err := conf.Db.Raw(
		`SELECT id, TIME_FORMAT(start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(over_time,'%H:%i:%s') AS over_time, price_id
		 FROM tab_price_time WHERE id = ?`, ctx.Query("id"),
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
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
	res := conf.Db.Exec(
		"INSERT INTO tab_price_time (start_time, over_time, price_id) VALUES (?, ?, ?)",
		req.StartTime, req.OverTime, req.PriceID,
	)
	ctx.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
}

// GetAll GET /priceTime/getAll 对齐 Java getAll：MyBatis-Plus 分页（无排序）。
func (c *PriceTimeController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.TabPriceTime{}).Count(&total)
	var rows []priceTimeRow
	conf.Db.Raw(
		`SELECT id, TIME_FORMAT(start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(over_time,'%H:%i:%s') AS over_time, price_id
		 FROM tab_price_time LIMIT ? OFFSET ?`,
		size, pageOffset(current, size),
	).Scan(&rows)
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
	var rows []priceTimeRow
	err := conf.Db.Raw(
		`SELECT tab_price_time.id, TIME_FORMAT(tab_price_time.start_time,'%H:%i:%s') AS start_time,
		        TIME_FORMAT(tab_price_time.over_time,'%H:%i:%s') AS over_time, tab_price_time.price_id
		 FROM tab_price_time
		 LEFT JOIN tab_price ON tab_price_time.price_id = tab_price.id
		 WHERE tab_price_time.over_time >= ? AND tab_price_time.start_time <= ?
		 ORDER BY (tab_price.total + tab_price.service) LIMIT 1`,
		start, end,
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
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
	type row struct {
		Total   decimal.Decimal
		Service decimal.Decimal
	}
	var rows []row
	err := conf.Db.Raw(
		`SELECT tab_price.total, tab_price.service
		 FROM tab_price_time
		 LEFT JOIN tab_price ON tab_price_time.price_id = tab_price.id
		 WHERE tab_price_time.start_time < CURTIME() AND tab_price_time.over_time >= CURTIME()
		 ORDER BY tab_price_time.over_time DESC LIMIT 1`,
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	r := rows[0]
	result := r.Total.Add(r.Service.Mul(decimal.NewFromFloat(0.8)))
	ctx.JSON(http.StatusOK, ResultSuccess(result))
}

// GetAllNow GET /priceTime/getAllNow 对齐 Java getAllNow：
// 当前时段（同上）的第一条的嵌套 price（PriceBean）。
func (c *PriceTimeController) GetAllNow(ctx *gin.Context) {
	var rows []priceDTO
	err := conf.Db.Raw(
		`SELECT tab_price.id, tab_price.total, tab_price.name, tab_price.service
		 FROM tab_price_time
		 LEFT JOIN tab_price ON tab_price_time.price_id = tab_price.id
		 WHERE tab_price_time.start_time < CURTIME() AND tab_price_time.over_time >= CURTIME()
		 ORDER BY tab_price_time.over_time DESC LIMIT 1`,
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(rows[0]))
}

// RegisterPricing 注册 ChargingPrice / ChargingPriceTime / price / priceTime 四组路由。
func RegisterPricing(r *gin.Engine) {
	chargingPrice := new(ChargingPriceController)
	chargingPriceTime := new(ChargingPriceTimeController)
	price := new(PriceController)
	priceTime := new(PriceTimeController)

	cp := r.Group("/ChargingPrice")
	{
		cp.GET("/getById", chargingPrice.GetById)
		cp.POST("/add", chargingPrice.Add)
		cp.GET("/getAll", chargingPrice.GetAll)
		cp.GET("/getMix", chargingPrice.GetMix)
		cp.GET("/getByTime", chargingPrice.GetByTime)
	}

	cpt := r.Group("/ChargingPriceTime")
	{
		cpt.GET("/getById", chargingPriceTime.GetById)
		cpt.POST("/add", chargingPriceTime.Add)
		cpt.GET("/getAll", chargingPriceTime.GetAll)
		cpt.GET("/getByTime", chargingPriceTime.GetByTime)
		cpt.GET("/getNow", chargingPriceTime.GetNow)
	}

	p := r.Group("/price")
	{
		p.GET("/getById", price.GetById)
		p.POST("/add", price.Add)
		p.GET("/getAll", price.GetAll)
		p.GET("/getMix", price.GetMix)
		p.GET("/getByTime", price.GetByTime)
	}

	pt := r.Group("/priceTime")
	{
		pt.GET("/getById", priceTime.GetById)
		pt.POST("/add", priceTime.Add)
		pt.GET("/getAll", priceTime.GetAll)
		pt.GET("/getByTime", priceTime.GetByTime)
		pt.GET("/getNow", priceTime.GetNow)
		pt.GET("/getAllNow", priceTime.GetAllNow)
	}
}
