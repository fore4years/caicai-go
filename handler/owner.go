package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
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

// saveNamedImage 读取 multipart 的指定字段并写入 tab_image，返回图片 id。
func saveNamedImage(ctx *gin.Context, field string) (int32, bool) {
	fh, err := ctx.FormFile(field)
	if err != nil {
		return 0, false
	}
	f, err := fh.Open()
	if err != nil {
		return 0, false
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return 0, false
	}
	img := model.TabImage{Image: data}
	if conf.Db.Create(&img).Error != nil {
		return 0, false
	}
	return img.ID, true
}

// nextPlaceSpacesCode 对齐 Java indexSerImpl.getNextSpacesCode：code + 5 位递增编号。
func nextPlaceSpacesCode(code string) string {
	if code == "" {
		return ""
	}
	var maxCode string
	conf.Db.Raw("SELECT MAX(placeid) FROM place_tbl WHERE placeid LIKE ?", code+"%").Scan(&maxCode)
	if maxCode == "" {
		maxCode = "0"
	}
	remainder := maxCode
	if len(maxCode) >= len(code) && maxCode[:len(code)] == code {
		remainder = maxCode[len(code):]
	}
	n, err := strconv.Atoi(remainder)
	if err != nil {
		return ""
	}
	return code + fmt.Sprintf("%05d", n+1)
}

// GetByIdPlace GET /wx/owner/getByIdPlace 对齐 Java indexSerImpl.getByIdPlace：按 placeid 查单个车位（返回列表）。
func (c *OwnerController) GetByIdPlace(ctx *gin.Context) {
	var place model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", ctx.Query("placeid")).First(&place).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess([]PlaceDtoDTO{}))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess([]PlaceDtoDTO{placeToPlaceDtoDTO(place)}))
}

// Save POST /wx/owner/save 对齐 Java indexCon.add + indexSerImpl.save：新增车位（证书图片可选，COS 上传省略）。
func (c *OwnerController) Save(ctx *gin.Context) {
	code := ctx.PostForm("code")
	openid := ctx.PostForm("openid")
	place := ctx.PostForm("place") // relatedBuilding
	province := ctx.PostForm("province")
	city := ctx.PostForm("city")
	district := ctx.PostForm("district")
	street := ctx.PostForm("street") // streer
	latitude := ctx.PostForm("latitude")
	longitude := ctx.PostForm("longitude")

	var imageID int32
	// Java：certificate == null ? null : imageService.add(certificate)；COS/磁盘上传省略，图片字节落 tab_image。
	if _, err := ctx.FormFile("file"); err == nil {
		imageID, _ = saveImageFromCtx(ctx)
	}

	spacesCode := nextPlaceSpacesCode(code)
	if spacesCode == "" {
		ctx.JSON(http.StatusOK, false)
		return
	}
	rec := model.PlaceTbl{
		Placeid:         spacesCode,
		Ownerid:         openid,
		Province:        province,
		City:            city,
		District:        district,
		Streer:          street,
		Longitude:       longitude,
		Latitude:        latitude,
		RelatedBuilding: place,
		State:           "待审核",
		ImageID:         imageID,
	}
	res := conf.Db.Select("placeid", "ownerid", "province", "city", "district", "streer", "longitude", "latitude", "relatedBuilding", "state", "image_id").
		Create(&rec)
	ctx.JSON(http.StatusOK, res.Error == nil)
}

// PlaceAddIdCardR POST /wx/owner/placeAdd_idCard_r 对齐 Java mySerImpl.placeAdd_r：回写身份证反面照。
// Java 用 saveFileUtil.save 落盘/COS，此处省略，图片字节存入 tab_image，以图片 id 字符串回写。
func (c *OwnerController) PlaceAddIdCardR(ctx *gin.Context) {
	id := ctx.PostForm("id")
	url := ""
	if _, err := ctx.FormFile("idCard_r"); err == nil {
		if imageID, ok := saveNamedImage(ctx, "idCard_r"); ok {
			url = strconv.Itoa(int(imageID))
		}
	}
	conf.Db.Model(&model.PlaceapplyTbl{}).Where("id = ?", id).Update("idCard_r", url)
	ctx.JSON(http.StatusOK, "ok")
}

// PlaceAddProve POST /wx/owner/placeAdd_prove 对齐 Java mySerImpl.placeAdd_p：回写产权证明。
// 同 placeAdd_idCard_r，文件存储省略，图片 id 字符串回写。
func (c *OwnerController) PlaceAddProve(ctx *gin.Context) {
	id := ctx.PostForm("id")
	url := ""
	if _, err := ctx.FormFile("prove"); err == nil {
		if imageID, ok := saveNamedImage(ctx, "prove"); ok {
			url = strconv.Itoa(int(imageID))
		}
	}
	conf.Db.Model(&model.PlaceapplyTbl{}).Where("id = ?", id).Update("prove", url)
	ctx.JSON(http.StatusOK, "ok")
}

// ownerMessageDTO 对应 Java domain.MessageBean。
type ownerMessageDTO struct {
	ID      int32  `json:"id,omitempty"`
	Ownerid string `json:"ownerid,omitempty"`
	Message string `json:"message,omitempty"`
	Msgtime string `json:"msgtime,omitempty"`
	Status  string `json:"status,omitempty"`
}

// GetMsgs GET /wx/owner/getmsgs 对齐 Java mySerImpl.getmsgs：查询业主消息并倒序。
func (c *OwnerController) GetMsgs(ctx *gin.Context) {
	var msgs []model.MessageTbl
	conf.Db.Where("ownerid = ?", ctx.Query("ownerid")).Find(&msgs)
	// Java Collections.reverse 倒序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	list := make([]ownerMessageDTO, 0, len(msgs))
	for _, m := range msgs {
		list = append(list, ownerMessageDTO{ID: m.ID, Ownerid: m.Ownerid, Message: m.Message, Msgtime: m.Msgtime, Status: m.Status})
	}
	ctx.JSON(http.StatusOK, list)
}
