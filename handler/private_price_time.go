package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
	"caicai-go/repository"
)

// PrivatePriceTimeController 对齐 Java controller.PrivatePriceTimeController（/PrivatePriceTime）。
type PrivatePriceTimeController struct{}

// privatePriceDTO 对应 Java PrivatePriceBean（private_charging_price_tbl）。
type privatePriceDTO struct {
	ID      int32           `json:"id"`
	Total   decimal.Decimal `json:"total,omitempty"`
	Name    string          `json:"name,omitempty"`
	Service decimal.Decimal `json:"service,omitempty"`
}

// privatePriceTimeDTO 对应 Java PrivatePriceTimeBean（tab_price_time / tab_price_time_{season}）。
type privatePriceTimeDTO struct {
	Price     *privatePriceDTO `json:"price,omitempty"`
	StartTime string           `json:"startTime,omitempty"`
	OverTime  string           `json:"overTime,omitempty"`
	ID        int32            `json:"id"`
	PriceID   int32            `json:"priceId,omitempty"`
}

// privatePriceTimeRow 时段电价查询行（时间用 TIME_FORMAT 格式化为字符串，避免 time.Time 解析歧义）。
type privatePriceTimeRow struct {
	StartTime string `gorm:"column:start_time"`
	OverTime  string `gorm:"column:over_time"`
	ID        int32  `gorm:"column:id"`
	PriceID   int32  `gorm:"column:price_id"`
}

const privatePriceTimeColumns = "id, TIME_FORMAT(start_time, '%H:%i:%s') AS start_time, TIME_FORMAT(over_time, '%H:%i:%s') AS over_time, price_id"

// privatePriceByID 按 id 查私桩电价。
func privatePriceByID(id int32) *privatePriceDTO {
	var p model.PrivateChargingPriceTbl
	if err := conf.Db.Where("id = ?", id).First(&p).Error; err != nil {
		return nil
	}
	return &privatePriceDTO{ID: p.ID, Total: p.Total, Name: p.Name, Service: p.Service}
}

// privatePriceTimeRowToDTO 将查询行转为 DTO，并关联电价。
func privatePriceTimeRowToDTO(r privatePriceTimeRow) privatePriceTimeDTO {
	dto := privatePriceTimeDTO{ID: r.ID, StartTime: r.StartTime, OverTime: r.OverTime, PriceID: r.PriceID}
	if p := privatePriceByID(r.PriceID); p != nil {
		dto.Price = p
	}
	return dto
}

// GetById GET /PrivatePriceTime/getById?id=
func (c *PrivatePriceTimeController) GetById(ctx *gin.Context) {
	var row privatePriceTimeRow
	if err := conf.Db.Raw("SELECT "+privatePriceTimeColumns+" FROM tab_price_time WHERE id = ?", queryInt(ctx, "id", 0)).Row().Scan(&row); err != nil || row.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privatePriceTimeRowToDTO(row)))
}

// Add POST /PrivatePriceTime/add
func (c *PrivatePriceTimeController) Add(ctx *gin.Context) {
	var req struct {
		ID        int32  `json:"id"`
		StartTime string `json:"startTime"`
		OverTime  string `json:"overTime"`
		PriceID   int32  `json:"priceId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if req.ID != 0 {
		res := conf.Db.Exec("UPDATE tab_price_time SET start_time = ?, over_time = ?, price_id = ? WHERE id = ?",
			req.StartTime, req.OverTime, req.PriceID, req.ID)
		ctx.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
		return
	}
	res := conf.Db.Exec("INSERT INTO tab_price_time (start_time, over_time, price_id) VALUES (?, ?, ?)",
		req.StartTime, req.OverTime, req.PriceID)
	ctx.JSON(http.StatusOK, ResultSuccess(res.Error == nil))
}

// GetAll GET /PrivatePriceTime/getAll?current=&size=
func (c *PrivatePriceTimeController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	conf.Db.Model(&model.TabPriceTime{}).Count(&total)

	var rows []privatePriceTimeRow
	conf.Db.Raw("SELECT "+privatePriceTimeColumns+" FROM tab_price_time ORDER BY id LIMIT ? OFFSET ?",
		size, pageOffset(current, size)).Scan(&rows)

	list := make([]privatePriceTimeDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privatePriceTimeRowToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetAllBySeason GET /PrivatePriceTime/getAllBySeason
func (c *PrivatePriceTimeController) GetAllBySeason(ctx *gin.Context) {
	table := "tab_price_time_" + repository.CurrentSeason()
	var rows []privatePriceTimeRow
	conf.Db.Raw("SELECT " + privatePriceTimeColumns + " FROM " + table).Scan(&rows)

	list := make([]privatePriceTimeDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privatePriceTimeRowToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetAllBySeasonAndPrivateOwner GET /PrivatePriceTime/getAllBySeasonAndPrivateOwner?privateOwnerId=
func (c *PrivatePriceTimeController) GetAllBySeasonAndPrivateOwner(ctx *gin.Context) {
	ownerID := ctx.Query("privateOwnerId")
	table := "tab_price_time_" + repository.CurrentSeason()
	var rows []privatePriceTimeRow
	conf.Db.Raw("SELECT " + privatePriceTimeColumns + " FROM " + table).Scan(&rows)

	// 桩主自己设置的服务费
	var ownerFee *decimal.Decimal
	if ownerID != "" {
		var pc model.PrivateChargingTbl
		if err := conf.Db.Where("id = ?", ownerID).First(&pc).Error; err == nil {
			ownerFee = &pc.Fee
		}
	}

	list := make([]privatePriceTimeDTO, 0, len(rows))
	for _, r := range rows {
		dto := privatePriceTimeRowToDTO(r)
		if dto.Price != nil && ownerFee != nil {
			dto.Price.Service = *ownerFee
		}
		list = append(list, dto)
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetByTime GET /PrivatePriceTime/getByTime?start=&end=（对齐 Java getMixByTime：最低价时段）
func (c *PrivatePriceTimeController) GetByTime(ctx *gin.Context) {
	start := ctx.Query("start")
	end := ctx.Query("end")
	var row privatePriceTimeRow
	err := conf.Db.Raw(`
		SELECT pt.id, TIME_FORMAT(pt.start_time, '%H:%i:%s') AS start_time,
		       TIME_FORMAT(pt.over_time, '%H:%i:%s') AS over_time, pt.price_id
		FROM tab_price_time pt
		LEFT JOIN private_charging_price_tbl p ON pt.price_id = p.id
		WHERE pt.over_time >= ? AND pt.start_time <= ?
		ORDER BY (p.total + p.service) LIMIT 1`, start, end).Row().Scan(&row)
	if err != nil || row.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privatePriceTimeRowToDTO(row)))
}

// GetNow GET /PrivatePriceTime/getNow（当前时段电价 total）
func (c *PrivatePriceTimeController) GetNow(ctx *gin.Context) {
	now := time.Now().Format("15:04:05")
	var total decimal.Decimal
	err := conf.Db.Raw(`
		SELECT p.total
		FROM tab_price_time pt
		LEFT JOIN private_charging_price_tbl p ON pt.price_id = p.id
		WHERE pt.start_time < ? AND pt.over_time >= ?
		ORDER BY pt.over_time DESC LIMIT 1`, now, now).Row().Scan(&total)
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(decimal.Zero))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(total))
}
