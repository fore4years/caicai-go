package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ PrivateChargingController（/privateCharging） ============

type PrivateChargingController struct{}

func (c *PrivateChargingController) Save(ctx *gin.Context) {
	openid := ctx.PostForm("openid")
	if openid == "" {
		openid = ctx.GetString("openid")
	}
	rec := model.PrivateChargingTbl{
		Openid:             openid,
		Name:               ctx.PostForm("name"),
		IDCard:             ctx.PostForm("idCard"),
		Phone:              ctx.PostForm("phone"),
		CommunityName:      ctx.PostForm("communityName"),
		CommunityAddress:   ctx.PostForm("communityAddress"),
		CommunitySpacesNum: ctx.PostForm("communitySpacesNum"),
		ProductType:        ctx.PostForm("productType"),
		ElectricityType:    ctx.PostForm("electricityType"),
		ProductID:          ctx.PostForm("chargingNum"),
		SpacesNum:          ctx.PostForm("spacesNum"),
		ShareTime:          ctx.PostForm("shareTime"),
		Status:             0,
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&rec).Error == nil))
}

func (c *PrivateChargingController) Register(ctx *gin.Context) {
	var rec model.PrivateChargingTbl
	if err := ctx.ShouldBindJSON(&rec); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&rec).Error == nil))
}

func (c *PrivateChargingController) DeleteById(ctx *gin.Context) {
	res := conf.Db.Where("id = ?", ctx.Param("id")).Delete(&model.PrivateChargingTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PrivateChargingController) GetById(ctx *gin.Context) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(p)))
}

func (c *PrivateChargingController) GetByOpenId(ctx *gin.Context) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("openid = ?", ctx.GetString("openid")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(p)))
}

func (c *PrivateChargingController) GetByOpenIdAndChargeId(ctx *gin.Context) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("openid = ? AND id = ?", ctx.GetString("openid"), ctx.Query("id")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(p)))
}

func (c *PrivateChargingController) GetListByOpenId(ctx *gin.Context) {
	var rows []model.PrivateChargingTbl
	conf.Db.Where("openid = ?", ctx.GetString("openid")).Find(&rows)
	list := make([]privateChargingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateChargingToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PrivateChargingController) GetByPidAndOpenId(ctx *gin.Context) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("product_id = ? AND openid = ?", ctx.Query("pid"), ctx.GetString("openid")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(p)))
}

func (c *PrivateChargingController) GetUserInfoByPid(ctx *gin.Context) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("product_id = ?", ctx.Query("pid")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(p)))
}

func (c *PrivateChargingController) GetAll(ctx *gin.Context) {
	var rows []model.PrivateChargingTbl
	conf.Db.Find(&rows)
	list := make([]privateChargingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateChargingToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PrivateChargingController) GetBySpacesNum(ctx *gin.Context) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("spaces_num = ?", ctx.Query("spacesNum")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(p)))
}

func (c *PrivateChargingController) Open(ctx *gin.Context) {
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id = ?", ctx.Query("id")).Update("one_click_opening", 1)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PrivateChargingController) UpdateShareTime(ctx *gin.Context) {
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id = ?", ctx.Query("id")).Updates(map[string]interface{}{
		"share_time":     ctx.Query("shareTime"),
		"exchange_place": ctx.Query("exchangePlace"),
		"overtime_fee":   ctx.Query("overtimeFee"),
	})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PrivateChargingController) Close(ctx *gin.Context) {
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id = ?", ctx.Query("id")).Update("one_click_opening", 0)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *PrivateChargingController) GetAllSharePlace(ctx *gin.Context) {
	var rows []model.PrivateChargingTbl
	conf.Db.Where("one_click_opening = 1").Find(&rows)
	list := make([]privateChargingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateChargingToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PrivateChargingController) GetSharePlaceByLocation(ctx *gin.Context) {
	c.GetAllSharePlace(ctx)
}

func (c *PrivateChargingController) GetSharePlaceByName(ctx *gin.Context) {
	var rows []model.PrivateChargingTbl
	conf.Db.Where("one_click_opening = 1 AND community_name LIKE ?", "%"+ctx.Query("name")+"%").Find(&rows)
	list := make([]privateChargingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateChargingToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PrivateChargingController) GetSharePlaceCountByName(ctx *gin.Context) {
	var n int64
	conf.Db.Model(&model.PrivateChargingTbl{}).Where("one_click_opening = 1 AND community_name LIKE ?", "%"+ctx.Query("name")+"%").Count(&n)
	ctx.JSON(http.StatusOK, ResultSuccess(n))
}

func (c *PrivateChargingController) SetInfoToPrivateUser(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

func (c *PrivateChargingController) SendNewRelocateCarMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

func (c *PrivateChargingController) UpdateByProductId(ctx *gin.Context) {
	var req struct {
		ProductID     string `json:"productId"`
		ShareTime     string `json:"shareTime"`
		CommunityName string `json:"communityName"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.ProductID == "" {
		ctx.JSON(http.StatusOK, ResultError(400, "充电桩编号不能为空"))
		return
	}
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("product_id = ?", req.ProductID).Updates(map[string]interface{}{
		"share_time":     req.ShareTime,
		"community_name": req.CommunityName,
	})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ SubscribeMessageController（/subscribeMessage）—— stub ============

type SubscribeMessageController struct{}

func (c *SubscribeMessageController) SelectRid(ctx *gin.Context) {
	ctx.String(http.StatusOK, "")
}
func (c *SubscribeMessageController) SendParkingMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendRelocateCarMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendParkingTicketMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendChargingMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendChargingSettleMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendChargingExceptionMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendOrderOverMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, true)
}
func (c *SubscribeMessageController) SendMoveCarMessage(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ HotelController 自助侧（/hotel）—— 简化 ============

type HotelSelfController struct{}

func (c *HotelSelfController) Add(ctx *gin.Context) {
	var h model.HotelTbl
	if err := ctx.ShouldBindJSON(&h); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	h.Openid = ctx.GetString("openid")
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&h).Error == nil))
}

func (c *HotelSelfController) GetByOpenid(ctx *gin.Context) {
	var rows []model.HotelTbl
	conf.Db.Where("openid = ?", ctx.GetString("openid")).Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

func (c *HotelSelfController) GetById(ctx *gin.Context) {
	var h model.HotelTbl
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&h).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(h))
}

func (c *HotelSelfController) OpenLock(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

func (c *HotelSelfController) CloseLock(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

func (c *HotelSelfController) GetStaffInfo(ctx *gin.Context) {
	var rows []model.HotelStaffTbl
	conf.Db.Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

// ============ ShopController（/wx 商户）—— 简化 ============

type ShopController struct{}

func (c *ShopController) GetShopByOpenId(ctx *gin.Context) {
	var s model.ShopTbl
	if err := conf.Db.Where("openid = ?", ctx.GetString("openid")).First(&s).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(s))
}

func (c *ShopController) GetShopInfoByShopId(ctx *gin.Context) {
	var s model.ShopTbl
	if err := conf.Db.Where("id = ?", ctx.Query("shopId")).First(&s).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(s))
}

func (c *ShopController) GetShopName(ctx *gin.Context) {
	var name string
	conf.Db.Model(&model.ShopTbl{}).Where("id = ?", ctx.Query("shopId")).Pluck("name", &name)
	ctx.JSON(http.StatusOK, ResultSuccess(name))
}

func (c *ShopController) SearchShop(ctx *gin.Context) {
	var rows []model.ShopTbl
	conf.Db.Where("name LIKE ?", "%"+ctx.Query("name")+"%").Find(&rows)
	ctx.JSON(http.StatusOK, ResultSuccess(rows))
}

// ============ OwnerController（/wx/owner、/wx）—— 简化 ============

type OwnerController struct{}

func (c *OwnerController) GetPlaceAll(ctx *gin.Context) {
	var places []model.PlaceTbl
	conf.Db.Where("ownerid = ?", ctx.GetString("openid")).Find(&places)
	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *OwnerController) GetPlace(ctx *gin.Context) {
	var p model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", ctx.Query("placeid")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(placeToDTO(p)))
}

func (c *OwnerController) GetOrders(ctx *gin.Context) {
	var orders []model.OrderTbl
	conf.Db.Where("openid = ?", ctx.GetString("openid")).Order("begin_time desc").Find(&orders)
	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, orderToDTO(o))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *OwnerController) PlaceAdd(ctx *gin.Context) {
	p := model.PlaceTbl{
		Placeid:  "PL" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Ownerid:  ctx.GetString("openid"),
		Province: ctx.PostForm("province"),
		City:     ctx.PostForm("city"),
		District: ctx.PostForm("district"),
		Streer:   ctx.PostForm("streer"),
		State:    "待审核",
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&p).Error == nil))
}

func (c *OwnerController) PlaceUpdate(ctx *gin.Context) {
	var p model.PlaceTbl
	if err := ctx.ShouldBindJSON(&p); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", p.Placeid).Updates(p)
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

func (c *OwnerController) LockAdd(ctx *gin.Context) {
	l := model.LockTbl{
		Lockid:  ctx.PostForm("lockid"),
		Lockmac: ctx.PostForm("lockmac"),
		Placeid: ctx.PostForm("placeid"),
		State:   "未使用",
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&l).Error == nil))
}

func (c *OwnerController) GetByLockMac(ctx *gin.Context) {
	var l model.LockTbl
	if err := conf.Db.Where("lockmac = ?", ctx.Query("lockmac")).First(&l).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(lockToDTO(l)))
}
