package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/model"
	"caicai-go/repository"
)

// PrivatePriceTimeController 对齐 Java controller.PrivatePriceTimeController（/PrivatePriceTime）。
type PrivatePriceTimeController struct{}

// privatePriceTimeDTO 对应 Java PrivatePriceTimeBean（tab_price_time / tab_price_time_{season}）。
type privatePriceTimeDTO struct {
	Price     *repository.PrivatePrice `json:"price,omitempty"`
	StartTime string                   `json:"startTime,omitempty"`
	OverTime  string                   `json:"overTime,omitempty"`
	ID        int32                    `json:"id"`
	PriceID   int32                    `json:"priceId,omitempty"`
}

// privatePriceByID 按 id 查私桩电价。
func privatePriceByID(id int32) *repository.PrivatePrice {
	p, err := repository.PrivatePriceByID(id)
	if err != nil {
		return nil
	}
	return &repository.PrivatePrice{ID: p.ID, Total: p.Total, Name: p.Name, Service: p.Service}
}

// privatePriceTimeRowToDTO 将查询行转为 DTO，并关联电价。
func privatePriceTimeRowToDTO(r repository.PrivatePriceTimeRow) privatePriceTimeDTO {
	dto := privatePriceTimeDTO{ID: r.ID, StartTime: r.StartTime, OverTime: r.OverTime, PriceID: r.PriceID}
	if p := privatePriceByID(r.PriceID); p != nil {
		dto.Price = p
	}
	return dto
}

// GetById GET /PrivatePriceTime/getById?id=
func (c *PrivatePriceTimeController) GetById(ctx *gin.Context) {
	row, err := repository.PrivatePriceTimeRowByID(queryInt(ctx, "id", 0))
	if err != nil || row.ID == 0 {
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
		err := repository.UpdatePrivatePriceTime(req.ID, req.StartTime, req.OverTime, req.PriceID)
		ctx.JSON(http.StatusOK, ResultSuccess(err == nil))
		return
	}
	err := repository.InsertPrivatePriceTime(req.StartTime, req.OverTime, req.PriceID)
	ctx.JSON(http.StatusOK, ResultSuccess(err == nil))
}

// GetAll GET /PrivatePriceTime/getAll?current=&size=
func (c *PrivatePriceTimeController) GetAll(ctx *gin.Context) {
	current, size := parsePage(ctx)
	var total int64
	repository.Count(&model.TabPriceTime{}, &total)

	rows := repository.PrivatePriceTimeRowsAll(pageOffset(current, size), size)

	list := make([]privatePriceTimeDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privatePriceTimeRowToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// GetAllBySeason GET /PrivatePriceTime/getAllBySeason
func (c *PrivatePriceTimeController) GetAllBySeason(ctx *gin.Context) {
	table := "tab_price_time_" + repository.CurrentSeason()
	rows := repository.PrivatePriceTimeRowsByTable(table)

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
	rows := repository.PrivatePriceTimeRowsByTable(table)

	// 桩主自己设置的服务费
	ownerFee := repository.PrivateChargingFeeByID(ownerID)

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
	row, err := repository.PrivatePriceTimeByTime(start, end)
	if err != nil || row.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privatePriceTimeRowToDTO(row)))
}

// GetNow GET /PrivatePriceTime/getNow（当前时段电价 total）
func (c *PrivatePriceTimeController) GetNow(ctx *gin.Context) {
	now := time.Now().Format("15:04:05")
	total, err := repository.PrivatePriceTotalNow(now)
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(decimal.Zero))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(total))
}
