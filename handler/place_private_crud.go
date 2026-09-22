package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ PlaceController（/place，业主车位） ============

type PlaceController struct{}

func (c *PlaceController) GetByOpenid(ctx *gin.Context) {
	var places []model.PlaceTbl
	conf.Db.Where("ownerid = ?", ctx.Param("openid")).Find(&places)
	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PlaceController) UpdateById(ctx *gin.Context) {
	var p model.PlaceTbl
	if err := ctx.ShouldBindJSON(&p); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", p.Placeid).Updates(p)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ PrivatePlaceController（/privatePlace） ============

type PrivatePlaceController struct{}

func (c *PrivatePlaceController) GetAll(ctx *gin.Context) {
	var rows []model.PrivatePlaceTbl
	conf.Db.Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *PrivatePlaceController) GetById(ctx *gin.Context) {
	var p model.PrivatePlaceTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(p))
}

func (c *PrivatePlaceController) Save(ctx *gin.Context) {
	var p model.PrivatePlaceTbl
	if err := ctx.ShouldBindJSON(&p); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "添加失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&p).Error == nil))
}

func (c *PrivatePlaceController) UpdateById(ctx *gin.Context) {
	var p model.PrivatePlaceTbl
	if err := ctx.ShouldBindJSON(&p); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	if id, err := strconv.Atoi(ctx.Param("id")); err == nil {
		p.ID = int32(id)
	}
	res := conf.Db.Model(&model.PrivatePlaceTbl{}).Where("id = ?", p.ID).Updates(p)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PrivatePlaceController) DeleteById(ctx *gin.Context) {
	res := conf.Db.Where("id = ?", ctx.Param("id")).Delete(&model.PrivatePlaceTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PrivatePlaceController) GetBySpacesCode(ctx *gin.Context) {
	var p model.PrivatePlaceTbl
	if err := conf.Db.Where("spaces_code = ?", ctx.Param("spacesCode")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(p))
}

func (c *PrivatePlaceController) GetByPlaceId(ctx *gin.Context) {
	var p model.PrivatePlaceTbl
	if err := conf.Db.Where("id = ?", ctx.Param("placeId")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(""))
		return
	}
	var gun model.TabChargingGun
	conf.Db.Where("id = ?", p.ChargingGunID).First(&gun)
	ctx.JSON(http.StatusOK, ResultSuccess(gun.ProductID))
}

func (c *PrivatePlaceController) GetByOrder(ctx *gin.Context) {
	var n int64
	conf.Db.Model(&model.OrderPrivateTbl{}).Where("orderid = ? AND is_private_user = 1", ctx.Param("orderId")).Count(&n)
	ctx.JSON(http.StatusOK, ResultSuccess(n > 0))
}

// ============ FreeChargingUserController（/freeChargingUsers） ============

type FreeChargingUserController struct{}

func (c *FreeChargingUserController) GetAll(ctx *gin.Context) {
	var rows []model.FreeChargingUsersTbl
	conf.Db.Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *FreeChargingUserController) GetById(ctx *gin.Context) {
	var u model.FreeChargingUsersTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&u).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(u))
}

func (c *FreeChargingUserController) GetByStationId(ctx *gin.Context) {
	var rows []model.FreeChargingUsersTbl
	conf.Db.Where("charging_station_id = ?", ctx.Param("stationId")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *FreeChargingUserController) GetByOpenId(ctx *gin.Context) {
	var u model.FreeChargingUsersTbl
	if err := conf.Db.Where("openid = ?", ctx.Param("openid")).First(&u).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(u))
}

func (c *FreeChargingUserController) Save(ctx *gin.Context) {
	var u model.FreeChargingUsersTbl
	if err := ctx.ShouldBindJSON(&u); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "添加失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&u).Error == nil))
}

func (c *FreeChargingUserController) Update(ctx *gin.Context) {
	var u model.FreeChargingUsersTbl
	if err := ctx.ShouldBindJSON(&u); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Model(&model.FreeChargingUsersTbl{}).Where("id = ?", ctx.Param("id")).Updates(u)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *FreeChargingUserController) UpdateByOpenId(ctx *gin.Context) {
	res := conf.Db.Model(&model.FreeChargingUsersTbl{}).
		Where("openid = ? AND charging_station_id = ?", ctx.Param("openid"), ctx.Param("stationId")).
		Update("status", 1)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *FreeChargingUserController) Delete(ctx *gin.Context) {
	res := conf.Db.Where("openid = ?", ctx.Param("openid")).Delete(&model.FreeChargingUsersTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ NeighborShareUserController（/neighborShareUser） ============

type NeighborShareUserController struct{}

func (c *NeighborShareUserController) Save(ctx *gin.Context) {
	var u model.NeighborShareUserTbl
	if err := ctx.ShouldBindJSON(&u); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&u).Error == nil))
}

func (c *NeighborShareUserController) Delete(ctx *gin.Context) {
	res := conf.Db.Where("id = ?", ctx.Param("id")).Delete(&model.NeighborShareUserTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *NeighborShareUserController) Update(ctx *gin.Context) {
	res := conf.Db.Model(&model.NeighborShareUserTbl{}).Where("id = ?", ctx.Param("id")).Update("status", 1)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *NeighborShareUserController) RefuseShareById(ctx *gin.Context) {
	res := conf.Db.Model(&model.NeighborShareUserTbl{}).Where("id = ?", ctx.Param("id")).Update("status", -1)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *NeighborShareUserController) CancelShareById(ctx *gin.Context) {
	res := conf.Db.Model(&model.NeighborShareUserTbl{}).Where("id = ?", ctx.Param("id")).Update("status", 0)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *NeighborShareUserController) GetById(ctx *gin.Context) {
	var u model.NeighborShareUserTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&u).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(u))
}

func (c *NeighborShareUserController) ListByOwner(ctx *gin.Context) {
	var rows []model.NeighborShareUserTbl
	conf.Db.Where("owner_id = ?", ctx.Param("ownerId")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *NeighborShareUserController) GetNoSharelistByOwner(ctx *gin.Context) {
	var rows []model.NeighborShareUserTbl
	conf.Db.Where("owner_id = ? AND status = 0", ctx.Param("ownerId")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *NeighborShareUserController) GetSharelistByOwner(ctx *gin.Context) {
	var rows []model.NeighborShareUserTbl
	conf.Db.Where("owner_id = ? AND status = 1", ctx.Param("ownerId")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *NeighborShareUserController) GetByPidAndOpenId(ctx *gin.Context) {
	var u model.NeighborShareUserTbl
	openid := ctx.GetString("openid")
	if err := conf.Db.Where("pid = ? AND openid = ?", ctx.Query("pid"), openid).First(&u).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(u))
}
