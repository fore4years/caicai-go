package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ OwnerController（/wx/owner、/wx） ============

type OwnerController struct{}

// GetPlaceAll 对齐 Java indexSer.getPlaceAll：查询所有车位。
func (c *OwnerController) GetPlaceAll(ctx *gin.Context) {
	var places []model.PlaceTbl
	conf.Db.Find(&places)
	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetPlace 对齐 Java indexSer.getPlace：按 ownerid 查车位并补充充电枪信息。
func (c *OwnerController) GetPlace(ctx *gin.Context) {
	ownerid := ctx.Query("ownerid")
	var rows []struct {
		model.PlaceTbl
		LockID string `gorm:"column:lockid"`
	}
	conf.Db.Raw(
		`SELECT p.*, l.lockid FROM place_tbl p
		 LEFT JOIN lock_tbl l ON p.placeid = l.placeid
		 WHERE ownerid = ? ORDER BY placeid ASC`, ownerid,
	).Scan(&rows)
	list := make([]PlaceDtoDTO, 0, len(rows))
	for _, r := range rows {
		dto := placeToPlaceDtoDTO(r.PlaceTbl)
		dto.ParkRate = r.ParkRate
		dto.LockID = r.LockID
		var cms []model.ChargeMessageTbl
		conf.Db.Where("placeid = ?", r.Placeid).Find(&cms)
		if len(cms) > 0 {
			dto.ChargeID = cms[len(cms)-1].Chargeid
			dto.Ischarge = 1
		}
		list = append(list, dto)
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetOrders 对齐 Java mySer.getOrder_owner（返回 List<OrderBean>，非 Result 包装）。
func (c *OwnerController) GetOrders(ctx *gin.Context) {
	var orders []model.OrderTbl
	conf.Db.Raw(
		`SELECT * FROM order_tbl
		 WHERE lockid IN (SELECT lockid FROM lock_tbl WHERE placeid IN (SELECT placeid FROM place_tbl WHERE ownerid = ?))
		   AND state = '已完成'
		 ORDER BY begin_time DESC`, ctx.Query("ownerid"),
	).Scan(&orders)
	list := make([]OrderDTO, 0, len(orders))
	for _, o := range orders {
		list = append(list, orderToDTO(o))
	}
	ctx.JSON(http.StatusOK, list)
}

// PlaceAdd 对齐 Java mySer.placeAdd：写入 placeapply + 消息。
func (c *OwnerController) PlaceAdd(ctx *gin.Context) {
	ownerid := ctx.PostForm("ownerid")
	name := ctx.PostForm("name")
	idNumber := ctx.PostForm("idNumber")
	province := ctx.PostForm("province")
	city := ctx.PostForm("city")
	district := ctx.PostForm("district")
	code := ctx.PostForm("code")
	streer := ctx.PostForm("streer")
	relatedBuilding := ctx.PostForm("relatedBuilding")
	carnumber := ctx.PostForm("carnumber")

	var existing model.PlaceapplyTbl
	if err := conf.Db.Where("name = ? AND code = ? AND streer = ? AND relatedBuilding = ? AND carnumber = ?",
		name, code, streer, relatedBuilding, carnumber).First(&existing).Error; err == nil {
		ctx.JSON(http.StatusOK, "noRepeat")
		return
	}

	id := simpleUUID()
	applyTime := sysTime()
	rec := model.PlaceapplyTbl{
		ID: id, Ownerid: ownerid, Name: name, IDNumber: idNumber,
		Province: province, City: city, District: district,
		Code: code, Streer: streer, RelatedBuilding: relatedBuilding,
		Carnumber: carnumber, ApplyTime: applyTime, State: "待审核",
	}
	if err := conf.Db.Create(&rec).Error; err != nil {
		ctx.JSON(http.StatusOK, id)
		return
	}

	msg := model.MessageTbl{
		Ownerid: ownerid,
		Message: name + "先生（女士），您好，您注册的车位：" + province + city + streer + relatedBuilding + ",正在努力审核中，请留意后续消息~",
		Msgtime: applyTime,
	}
	conf.Db.Create(&msg)

	ctx.JSON(http.StatusOK, id)
}

// PlaceUpdate 对齐 Java mySer.placeUpdate。
func (c *OwnerController) PlaceUpdate(ctx *gin.Context) {
	var p model.PlaceTbl
	if err := ctx.ShouldBindJSON(&p); err != nil {
		ctx.JSON(http.StatusOK, "ok")
		return
	}
	if p.Placeid == "" {
		ctx.JSON(http.StatusOK, "车位id不能为空！")
		return
	}
	var placeBean model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", p.Placeid).First(&placeBean).Error; err != nil {
		ctx.JSON(http.StatusOK, "车位不存在！")
		return
	}
	if placeBean.State == "使用中" {
		ctx.JSON(http.StatusOK, "车位使用中，暂时无法更新！")
		return
	}
	if p.ParkRate != "" {
		conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", p.Placeid).Updates(map[string]interface{}{
			"openTime":  p.OpenTime,
			"park_rate": p.ParkRate,
			"rate":      p.Rate,
		})
	} else {
		conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", p.Placeid).Updates(map[string]interface{}{
			"openTime": p.OpenTime,
			"rate":     p.Rate,
		})
	}
	ctx.JSON(http.StatusOK, "ok")
}

// LockAdd 对齐 Java lockAddSer.lockAdd。
func (c *OwnerController) LockAdd(ctx *gin.Context) {
	lockid := ctx.PostForm("lockid")
	lockmac := ctx.PostForm("lockmac")
	placeid := ctx.PostForm("placeid")
	if len(lockid) < 3 || lockid[:3] != "SCL" {
		ctx.JSON(http.StatusOK, "notOk")
		return
	}
	l := model.LockTbl{
		Lockid:    lockid,
		Lockmac:   lockmac,
		FixDate:   timePtr(time.Now()),
		Placeid:   placeid,
		State:     "正常",
		ProductID: ctx.PostForm("productId"),
	}
	if err := conf.Db.Create(&l).Error; err != nil {
		ctx.JSON(http.StatusOK, "notLockAdd")
		return
	}
	conf.Db.Model(&model.TabParkingSpace{}).Where("spaces_code = ?", placeid).Update("lock_id", lockid)
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Updates(map[string]interface{}{
		"longitude": ctx.PostForm("longitude"),
		"latitude":  ctx.PostForm("latitude"),
	})
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Update("state", "可使用")
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Update("rate", ctx.PostForm("rate"))
	conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", placeid).Update("openTime", ctx.PostForm("openTime"))
	ctx.JSON(http.StatusOK, "ok")
}

// GetByLockMac 对齐 Java lockAddSer.getByLockMac：返回锁的 lockmac。
func (c *OwnerController) GetByLockMac(ctx *gin.Context) {
	var l model.LockTbl
	if err := conf.Db.Where("lockid = ?", ctx.Query("lockid")).First(&l).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(l.Lockmac))
}
