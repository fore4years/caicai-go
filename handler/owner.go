package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ OwnerController（/wx/owner、/wx） ============

type OwnerController struct{}

// GetPlaceAll 对齐 Java indexSer.getPlaceAll：查询所有车位。
func (c *OwnerController) GetPlaceAll(ctx *gin.Context) {
	var places []model.PlaceTbl
	repository.FindAll(&places)
	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetPlace 对齐 Java indexSer.getPlace：按 ownerid 查车位并补充充电枪信息。
func (c *OwnerController) GetPlace(ctx *gin.Context) {
	ownerid := ctx.Query("ownerid")
	rows := repository.PlacesWithLockByOwnerid(ownerid)
	list := make([]PlaceDtoDTO, 0, len(rows))
	for _, r := range rows {
		dto := placeToPlaceDtoDTO(r.PlaceTbl)
		dto.ParkRate = r.ParkRate
		dto.LockID = r.LockID
		cms := repository.ChargeMessagesByPlaceid(r.Placeid)
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
	orders := repository.OrdersByOwnerid(ctx.Query("ownerid"))
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

	if repository.PlaceapplyExists(name, code, streer, relatedBuilding, carnumber) {
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
	if err := repository.Create(&rec).Error; err != nil {
		ctx.JSON(http.StatusOK, id)
		return
	}

	msg := model.MessageTbl{
		Ownerid: ownerid,
		Message: name + "先生（女士），您好，您注册的车位：" + province + city + streer + relatedBuilding + ",正在努力审核中，请留意后续消息~",
		Msgtime: applyTime,
	}
	repository.Create(&msg)

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
	placeBean, err := repository.PlaceByPlaceidDB(p.Placeid)
	if err != nil {
		ctx.JSON(http.StatusOK, "车位不存在！")
		return
	}
	if placeBean.State == "使用中" {
		ctx.JSON(http.StatusOK, "车位使用中，暂时无法更新！")
		return
	}
	if p.ParkRate != "" {
		repository.UpdatePlaceFields(p.Placeid, map[string]interface{}{
			"openTime":  p.OpenTime,
			"park_rate": p.ParkRate,
			"rate":      p.Rate,
		})
	} else {
		repository.UpdatePlaceFields(p.Placeid, map[string]interface{}{
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
	if err := repository.Create(&l).Error; err != nil {
		ctx.JSON(http.StatusOK, "notLockAdd")
		return
	}
	repository.UpdateParkingSpaceLockID(placeid, lockid)
	repository.UpdatePlaceFields(placeid, map[string]interface{}{
		"longitude": ctx.PostForm("longitude"),
		"latitude":  ctx.PostForm("latitude"),
	})
	repository.UpdatePlaceField(placeid, "state", "可使用")
	repository.UpdatePlaceField(placeid, "rate", ctx.PostForm("rate"))
	repository.UpdatePlaceField(placeid, "openTime", ctx.PostForm("openTime"))
	ctx.JSON(http.StatusOK, "ok")
}

// GetByLockMac 对齐 Java lockAddSer.getByLockMac：返回锁的 lockmac。
func (c *OwnerController) GetByLockMac(ctx *gin.Context) {
	l, err := repository.LockByLockid(ctx.Query("lockid"))
	if err != nil {
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
	if repository.Create(&img).Error != nil {
		return 0, false
	}
	return img.ID, true
}

// GetByIdPlace GET /wx/owner/getByIdPlace 对齐 Java indexSerImpl.getByIdPlace：按 placeid 查单个车位（返回列表）。
func (c *OwnerController) GetByIdPlace(ctx *gin.Context) {
	place, err := repository.PlaceByPlaceidDB(ctx.Query("placeid"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess([]PlaceDtoDTO{}))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess([]PlaceDtoDTO{placeToPlaceDtoDTO(*place)}))
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

	spacesCode := repository.NextPlaceSpacesCode(code)
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
	err := repository.CreateFields(&rec, []string{"placeid", "ownerid", "province", "city", "district", "streer", "longitude", "latitude", "relatedBuilding", "state", "image_id"})
	ctx.JSON(http.StatusOK, err == nil)
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
	repository.UpdatePlaceapplyField(id, "idCard_r", url)
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
	repository.UpdatePlaceapplyField(id, "prove", url)
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
	msgs := repository.MessagesByOwnerid(ctx.Query("ownerid"))
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
