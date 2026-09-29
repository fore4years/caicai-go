package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

func sysTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// formBool 将表单字符串解析为 bool（对齐 Java Integer 0/1）。
func formBool(s string) bool {
	return s == "1" || s == "true"
}

// saveImageFromCtx 读取 multipart 的 file 字段并写入 tab_image，返回图片 id。
func saveImageFromCtx(ctx *gin.Context) (int32, bool) {
	fh, err := ctx.FormFile("file")
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

// sysTimeDiffMs 计算两个 "yyyy-MM-dd HH:mm:ss" 之间的毫秒差（end - start）。
func sysTimeDiffMs(start, end string) int {
	t1, err1 := time.ParseInLocation("2006-01-02 15:04:05", start, time.Local)
	t2, err2 := time.ParseInLocation("2006-01-02 15:04:05", end, time.Local)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(t2.Sub(t1).Milliseconds())
}

// ============ PrivateChargingController（/privateCharging） ============

// privateShareChargingDTO 对应 Java domain.vo.PrivateShareChargingVo。
type privateShareChargingDTO struct {
	ID                 int32           `json:"id,omitempty"`
	CommunityName      string          `json:"communityName,omitempty"`
	CommunitySpacesNum string          `json:"communitySpacesNum,omitempty"`
	ProductType        string          `json:"productType,omitempty"`
	ElectricityType    string          `json:"electricityType,omitempty"`
	ShareTime          string          `json:"shareTime,omitempty"`
	ExchangePlace      bool            `json:"exchangePlace,omitempty"`
	Fee                decimal.Decimal `json:"fee,omitempty"`
	ImageID            int32           `json:"imageId,omitempty"`
	DistanceKm         float64         `json:"distanceKm,omitempty"`
}

func privateShareChargingToDTO(p model.PrivateChargingTbl) privateShareChargingDTO {
	return privateShareChargingDTO{
		ID:                 p.ID,
		CommunityName:      p.CommunityName,
		CommunitySpacesNum: p.CommunitySpacesNum,
		ProductType:        p.ProductType,
		ElectricityType:    p.ElectricityType,
		ShareTime:          p.ShareTime,
		ExchangePlace:      p.ExchangePlace,
		Fee:                p.Fee,
		ImageID:            p.ImageID,
	}
}

// privateChargingDetailDTO 对应 Java getUserAndPlaceInfoById 的 PrivateChargingVo（pc.* + 车位信息）。
type privateChargingDetailDTO struct {
	privateChargingDTO
	Point         string `json:"point,omitempty"`
	Place         string `json:"place,omitempty"`
	ChargingGunID int32  `json:"chargingGunId,omitempty"`
}

// privateChargingPwmDTO 对应 Java getAllWithPwm 返回（bean + pwm）。
type privateChargingPwmDTO struct {
	privateChargingDTO
	Pwm int32 `json:"pwm,omitempty"`
}

// privateChargingVoReq 对应 Java PrivateChargingVo（save 表单字段）。
type privateChargingVoReq struct {
	Name               string
	IDCard             string
	Phone              string
	Address            string
	SpacesNum          string
	ChargingNum        string
	IsUpdateImg        string
	Fee                decimal.Decimal
	CommunityName      string
	CommunityAddress   string
	CommunitySpacesNum string
	ProductType        string
	ElectricityType    string
	ShareTime          string
	OvertimeFee        bool
	OfficialUse        bool
}

func privateChargingVoFromForm(ctx *gin.Context) *privateChargingVoReq {
	fee, _ := decimal.NewFromString(ctx.PostForm("fee"))
	return &privateChargingVoReq{
		Name:               ctx.PostForm("name"),
		IDCard:             ctx.PostForm("idCard"),
		Phone:              ctx.PostForm("phone"),
		Address:            ctx.PostForm("address"),
		SpacesNum:          ctx.PostForm("spacesNum"),
		ChargingNum:        ctx.PostForm("chargingNum"),
		IsUpdateImg:        ctx.PostForm("isUpdateImg"),
		Fee:                fee,
		CommunityName:      ctx.PostForm("communityName"),
		CommunityAddress:   ctx.PostForm("communityAddress"),
		CommunitySpacesNum: ctx.PostForm("communitySpacesNum"),
		ProductType:        ctx.PostForm("productType"),
		ElectricityType:    ctx.PostForm("electricityType"),
		ShareTime:          ctx.PostForm("shareTime"),
		OvertimeFee:        formBool(ctx.PostForm("overtimeFee")),
		OfficialUse:        formBool(ctx.PostForm("officialUse")),
	}
}

// privateRegisterReq 对应 Java PrivateRegisterVo。
type privateRegisterReq struct {
	Name        string  `json:"name"`
	IDCard      string  `json:"idCard"`
	Phone       string  `json:"phone"`
	Address     string  `json:"address"`
	SpacesNum   string  `json:"spacesNum"`
	ChargingNum string  `json:"chargingNum"`
	Pid         string  `json:"pid"`
	Imei        string  `json:"imei"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Code        string  `json:"code"`
	HasCpLine   int32   `json:"hasCpLine"`
}

type PrivateChargingController struct{}

// Save POST /privateCharging/save 对齐 Java PrivateChargingBeanServiceImpl.add。
func (c *PrivateChargingController) Save(ctx *gin.Context) {
	openid := ctx.GetString("openid")
	if openid == "" {
		openid = ctx.PostForm("openid")
	}

	vo := privateChargingVoFromForm(ctx)
	if _, err := ctx.FormFile("file"); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess("上传的文件不能为空"))
		return
	}
	if openid == "" {
		ctx.JSON(http.StatusOK, ResultSuccess("用户未登录或会话已过期"))
		return
	}
	spacesNum := vo.SpacesNum
	if spacesNum == "" {
		ctx.JSON(http.StatusOK, ResultSuccess("车位编号不能为空"))
		return
	}

	var pp model.PrivatePlaceTbl
	if err := conf.Db.Where("spaces_code = ?", spacesNum).First(&pp).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess("无此车位编号, 请重新输入"))
		return
	}
	if existing := privateChargingBySpacesNum(spacesNum); existing != nil && existing.Openid != openid {
		ctx.JSON(http.StatusOK, ResultSuccess("此车位编号已存在, 请重新输入"))
		return
	}

	bean := model.PrivateChargingTbl{
		Openid:             openid,
		Name:               vo.Name,
		IDCard:             vo.IDCard,
		Phone:              vo.Phone,
		CommunityName:      vo.CommunityName,
		CommunityAddress:   vo.CommunityAddress,
		CommunitySpacesNum: vo.CommunitySpacesNum,
		ProductType:        vo.ProductType,
		ElectricityType:    vo.ElectricityType,
		ProductID:          vo.ChargingNum,
		SpacesNum:          spacesNum,
		ShareTime:          vo.ShareTime,
		Fee:                vo.Fee,
		OvertimeFee:        vo.OvertimeFee,
		OfficialUse:        vo.OfficialUse,
	}

	var beanExisting model.PrivateChargingTbl
	err := conf.Db.Where("openid = ? AND spaces_num = ?", openid, spacesNum).First(&beanExisting).Error
	if err == nil {
		if vo.IsUpdateImg == "true" {
			imageID, ok := saveImageFromCtx(ctx)
			if !ok {
				ctx.JSON(http.StatusOK, ResultSuccess("图片上传失败"))
				return
			}
			bean.ImageID = imageID
			if beanExisting.ImageID != 0 {
				conf.Db.Where("id = ?", beanExisting.ImageID).Delete(&model.TabImage{})
			}
		} else {
			bean.ImageID = beanExisting.ImageID
		}
		conf.Db.Model(&model.PrivateChargingTbl{}).Where("openid = ? AND spaces_num = ?", openid, spacesNum).Updates(bean)
	} else {
		if vo.IsUpdateImg == "true" {
			imageID, ok := saveImageFromCtx(ctx)
			if !ok {
				ctx.JSON(http.StatusOK, ResultSuccess("图片上传失败"))
				return
			}
			bean.ImageID = imageID
		}
		if conf.Db.Create(&bean).Error != nil {
			ctx.JSON(http.StatusOK, ResultSuccess("添加失败"))
			return
		}
	}
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// Register POST /privateCharging/register 对齐 Java PrivateChargingBeanServiceImpl.register。
func (c *PrivateChargingController) Register(ctx *gin.Context) {
	var req privateRegisterReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	openid := ctx.GetString("openid")

	var spacesNum string
	product := productByPid(req.Pid)
	if product == nil {
		// 创建产品
		p := model.TabProduct{Pid: req.Pid, Environment: "私桩", IsTwicecar: 0}
		if len(req.Imei) > 12 {
			p.Imei = req.Imei
		} else {
			p.Nid = req.Imei
		}
		var gw model.TabGateway
		conf.Db.Where("mac = ?", req.Imei).First(&gw)
		if gw.ID == 0 {
			gw = model.TabGateway{Mac: req.Imei, InstallPlace: "私桩"}
			conf.Db.Create(&gw)
			conf.Db.Where("mac = ?", req.Imei).First(&gw)
		}
		p.GatewayID = gw.ID
		conf.Db.Create(&p)

		// 创建充电枪
		var gun model.TabChargingGun
		conf.Db.Where("product_id = ?", req.Pid).First(&gun)
		if gun.ID == 0 {
			gun = model.TabChargingGun{Direction: "D", MeterValue: 0, ProductID: req.Pid}
			conf.Db.Create(&gun)
			conf.Db.Where("product_id = ?", req.Pid).First(&gun)
		}
		spacesNum = insertPrivatePlace(req, gun.ID)
	}

	rec := model.PrivateChargingTbl{
		Openid:    openid,
		Name:      req.Name,
		IDCard:    req.IDCard,
		Phone:     req.Phone,
		ProductID: req.Pid,
		Status:    0,
		HasCpLine: req.HasCpLine,
	}

	if spacesNum == "" {
		var gun model.TabChargingGun
		conf.Db.Where("product_id = ?", req.Pid).First(&gun)
		var pp model.PrivatePlaceTbl
		conf.Db.Where("charging_gun_id = ?", gun.ID).First(&pp)
		spacesNum = pp.SpacesCode
	}
	rec.SpacesNum = spacesNum

	var existing model.PrivateChargingTbl
	if err := conf.Db.Where("spaces_num = ?", spacesNum).First(&existing).Error; err == nil {
		ctx.JSON(http.StatusOK, ResultError(500, "此车位编号已被人注册"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(conf.Db.Create(&rec).Error == nil))
}

// insertPrivatePlace 对齐 Java PrivatePlaceServiceImpl.insert，返回生成的车位编号。
func insertPrivatePlace(req privateRegisterReq, gunID int32) string {
	if req.Code == "" {
		return ""
	}
	spacesCode := nextPrivateSpacesCode(req.Code)
	if spacesCode == "" {
		return ""
	}
	var existing model.PrivatePlaceTbl
	if err := conf.Db.Where("charging_gun_id = ?", gunID).First(&existing).Error; err == nil {
		return ""
	}
	pointWKT := fmt.Sprintf("POINT(%f %f)", req.Longitude, req.Latitude)
	if err := conf.Db.Exec(
		"INSERT INTO private_place_tbl (point, place, openid, charging_gun_id, spaces_code) VALUES (ST_GeomFromText(?), ?, ?, ?, ?)",
		pointWKT, req.Address, "", gunID, spacesCode,
	).Error; err != nil {
		return ""
	}
	return spacesCode
}

// nextPrivateSpacesCode 对齐 Java getNextSpacesCode：code + 5 位递增编号。
func nextPrivateSpacesCode(code string) string {
	if code == "" {
		return ""
	}
	var maxCode string
	conf.Db.Raw("SELECT MAX(spaces_code) FROM private_place_tbl WHERE spaces_code LIKE ?", code+"%").Scan(&maxCode)
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

func (c *PrivateChargingController) DeleteById(ctx *gin.Context) {
	res := conf.Db.Where("id = ?", ctx.Param("id")).Delete(&model.PrivateChargingTbl{})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// GetById 对齐 Java getUserAndPlaceInfoById：私桩信息 + 车位地址/充电枪。
func (c *PrivateChargingController) GetById(ctx *gin.Context) {
	var p model.PrivateChargingTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	dto := privateChargingDetailDTO{privateChargingDTO: privateChargingToDTO(p)}
	var pp struct {
		Point         string `gorm:"column:point"`
		Place         string `gorm:"column:place"`
		ChargingGunID int32  `gorm:"column:charging_gun_id"`
	}
	if err := conf.Db.Raw(
		"SELECT ST_AsText(point) AS point, place, charging_gun_id FROM private_place_tbl WHERE spaces_code = ?", p.SpacesNum,
	).Scan(&pp).Error; err == nil {
		dto.Point = pp.Point
		dto.Place = pp.Place
		dto.ChargingGunID = pp.ChargingGunID
	}
	ctx.JSON(http.StatusOK, ResultSuccess(dto))
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

// GetByPidAndOpenId 对齐 Java getByPidAndOpenId：经 private_place + charging_gun 关联。
func (c *PrivateChargingController) GetByPidAndOpenId(ctx *gin.Context) {
	var p model.PrivateChargingTbl
	err := conf.Db.Raw(
		`SELECT private_charging_tbl.* FROM private_charging_tbl
		 LEFT JOIN private_place_tbl pp ON private_charging_tbl.spaces_num = pp.spaces_code
		 LEFT JOIN tab_charging_gun tcg ON pp.charging_gun_id = tcg.id
		 WHERE tcg.product_id = ? AND private_charging_tbl.openid = ?`,
		ctx.Query("pid"), ctx.GetString("openid"),
	).Scan(&p).Error
	if err != nil || p.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(p)))
}

// GetUserInfoByPid 对齐 Java getUserInfoByPid。
func (c *PrivateChargingController) GetUserInfoByPid(ctx *gin.Context) {
	var p model.PrivateChargingTbl
	err := conf.Db.Raw(
		`SELECT private_charging_tbl.* FROM private_charging_tbl
		 LEFT JOIN private_place_tbl pp ON private_charging_tbl.spaces_num = pp.spaces_code
		 LEFT JOIN tab_charging_gun tcg ON pp.charging_gun_id = tcg.id
		 WHERE tcg.product_id = ?`,
		ctx.Query("pid"),
	).Scan(&p).Error
	if err != nil || p.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(p)))
}

// GetAll 对齐 Java getAllWithPwm：私桩信息 + pwm。
func (c *PrivateChargingController) GetAll(ctx *gin.Context) {
	var rows []struct {
		model.PrivateChargingTbl
		Pwm int32 `gorm:"column:pwm"`
	}
	conf.Db.Raw(
		`SELECT private_charging_tbl.*, tab_charging_gun.pwm
		 FROM private_charging_tbl
		 LEFT JOIN tab_charging_gun ON private_charging_tbl.product_id = tab_charging_gun.product_id`,
	).Scan(&rows)
	list := make([]privateChargingPwmDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateChargingPwmDTO{privateChargingDTO: privateChargingToDTO(r.PrivateChargingTbl), Pwm: r.Pwm})
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
	overtimeFee := 0
	if ctx.Query("overtimeFee") != "" {
		overtimeFee, _ = strconv.Atoi(ctx.Query("overtimeFee"))
	}
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("id = ?", ctx.Query("id")).Updates(map[string]interface{}{
		"share_time":     ctx.Query("shareTime"),
		"exchange_place": ctx.Query("exchangePlace"),
		"overtime_fee":   overtimeFee,
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

// GetSharePlaceByLocation 对齐 Java getSharePlaceByLocation（3km 内共享车位）。
func (c *PrivateChargingController) GetSharePlaceByLocation(ctx *gin.Context) {
	latitude := queryFloat(ctx, "latitude")
	longitude := queryFloat(ctx, "longitude")
	var rows []struct {
		model.PrivateChargingTbl
		DistanceKm float64 `gorm:"column:distance_km"`
	}
	conf.Db.Raw(
		`SELECT private_charging_tbl.id, private_charging_tbl.community_spaces_num, private_charging_tbl.share_time,
		        private_charging_tbl.fee, private_charging_tbl.product_type, private_charging_tbl.image_id,
		        private_charging_tbl.electricity_type, private_charging_tbl.product_id, private_charging_tbl.community_name,
		        private_charging_tbl.exchange_place,
		        ROUND(6371 * ACOS(COS(RADIANS(?)) * COS(RADIANS(ST_Y(private_place_tbl.point)))
		              * COS(RADIANS(ST_X(private_place_tbl.point)) - RADIANS(?)) + SIN(RADIANS(?))
		              * SIN(RADIANS(ST_Y(private_place_tbl.point)))),2) AS distance_km
		 FROM private_charging_tbl
		 LEFT JOIN private_place_tbl ON private_place_tbl.spaces_code = private_charging_tbl.spaces_num
		 WHERE private_place_tbl.point IS NOT NULL
		   AND private_charging_tbl.one_click_opening = 1
		   AND private_charging_tbl.status = 1
		 HAVING distance_km < 3
		 ORDER BY distance_km ASC`,
		latitude, longitude, latitude,
	).Scan(&rows)
	list := make([]privateShareChargingDTO, 0, len(rows))
	for _, r := range rows {
		dto := privateShareChargingToDTO(r.PrivateChargingTbl)
		dto.DistanceKm = r.DistanceKm
		list = append(list, dto)
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetSharePlaceByName 对齐 Java getSharePlaceByName：小区名等值匹配。
func (c *PrivateChargingController) GetSharePlaceByName(ctx *gin.Context) {
	var rows []model.PrivateChargingTbl
	conf.Db.Where("community_name = ? AND one_click_opening = 1 AND status = 1", ctx.Query("name")).Find(&rows)
	list := make([]privateShareChargingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateShareChargingToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PrivateChargingController) GetSharePlaceCountByName(ctx *gin.Context) {
	var n int64
	conf.Db.Model(&model.PrivateChargingTbl{}).Where("community_name = ? AND one_click_opening = 1 AND status = 1", ctx.Query("name")).Count(&n)
	ctx.JSON(http.StatusOK, ResultSuccess(int(n)))
}

func (c *PrivateChargingController) SetInfoToPrivateUser(ctx *gin.Context) {
	// 微信订阅消息推送省略。
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// SendNewRelocateCarMessage 对齐 Java sendNewRelocateCarMessage：更新挪车通知时间。
func (c *PrivateChargingController) SendNewRelocateCarMessage(ctx *gin.Context) {
	var req struct {
		OrderID       string `json:"orderId"`
		Openid        string `json:"openid"`
		IsPrivateUser string `json:"isPrivateUser"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	var order model.OrderTbl
	if err := conf.Db.Where("orderid = ?", req.OrderID).First(&order).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", req.OrderID).Update("neighbor_relocate_time", time.Now())
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// UpdateByProductId 对齐 Java updateByProductId。
func (c *PrivateChargingController) UpdateByProductId(ctx *gin.Context) {
	var req struct {
		ProductID          string          `json:"productId"`
		Name               string          `json:"name"`
		IDCard             string          `json:"idCard"`
		Phone              string          `json:"phone"`
		CommunityName      string          `json:"communityName"`
		CommunityAddress   string          `json:"communityAddress"`
		CommunitySpacesNum string          `json:"communitySpacesNum"`
		ProductType        string          `json:"productType"`
		ElectricityType    string          `json:"electricityType"`
		ShareTime          string          `json:"shareTime"`
		ExchangePlace      bool            `json:"exchangePlace"`
		OvertimeFee        bool            `json:"overtimeFee"`
		SpacesNum          string          `json:"spacesNum"`
		Fee                decimal.Decimal `json:"fee"`
		OfficialUse        bool            `json:"officialUse"`
		OneClickOpening    int32           `json:"oneClickOpening"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.ProductID == "" {
		ctx.JSON(http.StatusOK, ResultError(400, "充电桩编号不能为空"))
		return
	}
	var existing model.PrivateChargingTbl
	if err := conf.Db.Where("product_id = ?", req.ProductID).First(&existing).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "未找到对应的私桩信息"))
		return
	}
	res := conf.Db.Model(&model.PrivateChargingTbl{}).Where("product_id = ?", req.ProductID).Updates(map[string]interface{}{
		"name":                 req.Name,
		"id_card":              req.IDCard,
		"phone":                req.Phone,
		"community_name":       req.CommunityName,
		"community_address":    req.CommunityAddress,
		"community_spaces_num": req.CommunitySpacesNum,
		"product_type":         req.ProductType,
		"electricity_type":     req.ElectricityType,
		"share_time":           req.ShareTime,
		"exchange_place":       req.ExchangePlace,
		"overtime_fee":         req.OvertimeFee,
		"spaces_num":           req.SpacesNum,
		"fee":                  req.Fee,
		"official_use":         req.OfficialUse,
		"one_click_opening":    req.OneClickOpening,
	})
	ctx.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// ============ SubscribeMessageController（/subscribeMessage） ============

type SubscribeMessageController struct{}

// SelectRid 查询 rid（微信接口，省略外呼，返回空串）。
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

// SendMoveCarMessage 对齐 Java sendMoveCarMessage：更新 move_car_time。
func (c *SubscribeMessageController) SendMoveCarMessage(ctx *gin.Context) {
	var req struct {
		CarNumber string `json:"carNumber"`
		Phone     string `json:"phone"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	var user model.UserTbl
	if err := conf.Db.Where("phone = ?", req.Phone).First(&user).Error; err != nil || user.Openid == "" {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	var latest model.OrderTbl
	if err := conf.Db.Where("openid = ? AND consumption_type = ? AND state = ?", user.Openid, "邻享充电", "使用中").
		Order("begin_time desc").First(&latest).Error; err == nil {
		conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", latest.Orderid).Update("move_car_time", time.Now())
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
}

// ============ HotelController 自助侧（/hotel） ============

type HotelSelfController struct{}

// Add 对齐 Java HotelServiceImpl.add：更新已有酒店（要求已存在）。
func (c *HotelSelfController) Add(ctx *gin.Context) {
	var h model.HotelTbl
	if err := ctx.ShouldBindJSON(&h); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(0))
		return
	}
	var existing model.HotelTbl
	if err := conf.Db.Where("id = ?", h.ID).First(&existing).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "酒店封面未上传"))
		return
	}
	h.CreatTime = time.Now()
	res := conf.Db.Model(&model.HotelTbl{}).Where("id = ?", h.ID).Updates(h)
	ctx.JSON(http.StatusOK, ResultSuccess(int(res.RowsAffected)))
}

func (c *HotelSelfController) GetByOpenid(ctx *gin.Context) {
	var rows []model.HotelTbl
	conf.Db.Where("openid = ?", ctx.GetString("openid")).Find(&rows)
	list := make([]hotelDTO, 0, len(rows))
	for _, h := range rows {
		list = append(list, hotelToDTO(h))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *HotelSelfController) GetById(ctx *gin.Context) {
	var h model.HotelTbl
	if err := conf.Db.Where("id = ?", ctx.Query("id")).First(&h).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(hotelToDTO(h)))
}

// OpenLock 对齐 Java openLock：更新 hotel_product_tbl 使用状态（硬件指令省略）。
func (c *HotelSelfController) OpenLock(ctx *gin.Context) {
	lockID := ctx.Query("lockId")
	var hp model.HotelProductTbl
	if err := conf.Db.Where("productId = ?", lockID).First(&hp).Error; err == nil {
		now := sysTime()
		conf.Db.Model(&model.HotelProductTbl{}).Where("productId = ?", lockID).Updates(map[string]interface{}{
			"useState":   "1",
			"startTime":  now,
			"updateTime": now,
		})
	}
	ctx.JSON(http.StatusOK, ResultSuccess(1))
}

// CloseLock 对齐 Java closeLock：更新使用状态并累计使用时长。
func (c *HotelSelfController) CloseLock(ctx *gin.Context) {
	lockID := ctx.Query("lockId")
	var hp model.HotelProductTbl
	if err := conf.Db.Where("productId = ?", lockID).First(&hp).Error; err == nil {
		endTime := sysTime()
		oldUseTime := 0
		if hp.UseTime != "" {
			oldUseTime, _ = strconv.Atoi(hp.UseTime)
		}
		currentUseTimeMs := sysTimeDiffMs(hp.StartTime, endTime)
		currentUseTimeMinutes := (currentUseTimeMs + 59999) / 60000
		conf.Db.Model(&model.HotelProductTbl{}).Where("productId = ?", lockID).Updates(map[string]interface{}{
			"useState":   "0",
			"updateTime": endTime,
			"endTime":    endTime,
			"useTime":    strconv.Itoa(oldUseTime + currentUseTimeMinutes),
		})
	}
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// GetStaffInfo 对齐 Java getStaffInfo：按 hotelId + openId 查询单个员工。
func (c *HotelSelfController) GetStaffInfo(ctx *gin.Context) {
	var s model.HotelStaffTbl
	if err := conf.Db.Where("hotelId = ? AND openId = ?", ctx.Query("hotelId"), ctx.Query("openId")).First(&s).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(s))
}

// ============ ShopController（/wx 商户） ============

// shopDTO 对应 Java domain.Shop（驼峰）。
type shopDTO struct {
	ID         int32  `json:"id,omitempty"`
	Icon       string `json:"icon,omitempty"`
	ShopID     string `json:"shopId,omitempty"`
	Name       string `json:"name,omitempty"`
	Address    string `json:"address,omitempty"`
	WorkState  string `json:"workState,omitempty"`
	WorkTime   string `json:"workTime,omitempty"`
	CreatTime  string `json:"creatTime,omitempty"`
	UpdateTime string `json:"updateTime,omitempty"`
}

func shopToDTO(s model.ShopTbl) shopDTO {
	return shopDTO{
		ID: s.ID, Icon: s.Icon, ShopID: s.ShopID, Name: s.Name, Address: s.Address,
		WorkState: s.WorkState, WorkTime: s.WorkTime, CreatTime: s.CreatTime, UpdateTime: s.UpdateTime,
	}
}

// shopInfoDTO 对应 Java dto.ShopDto。
type shopInfoDTO struct {
	ShopID    string `json:"shopId,omitempty"`
	URL       string `json:"url,omitempty"`
	Name      string `json:"name,omitempty"`
	State     string `json:"state,omitempty"`
	Address   string `json:"address,omitempty"`
	WorkTime  string `json:"workTime,omitempty"`
	CheweiNum string `json:"cheweiNum,omitempty"`
}

// shopToShopInfo 对应 Java shopToShopDto。
func shopToShopInfo(shop model.ShopTbl) shopInfoDTO {
	state := "休息中"
	if shop.WorkState == "1" {
		state = "营业中"
	}
	address := ""
	cheweiNum := "0"
	var place model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", shop.Address).First(&place).Error; err == nil {
		address = place.Province + place.City + place.District + place.Streer + shop.Name
		var locks []model.LockTbl
		conf.Db.Where("placeid = ?", place.Placeid).Find(&locks)
		num := 0
		for _, l := range locks {
			var lui model.LockUseTbl
			if conf.Db.Where("lockid = ?", l.Lockid).First(&lui).Error == nil && lui.UseState == "0" {
				num++
			}
		}
		cheweiNum = strconv.Itoa(num)
	}
	return shopInfoDTO{
		ShopID: shop.ShopID, URL: shop.Icon, Name: shop.Name, State: state,
		Address: address, WorkTime: shop.WorkTime, CheweiNum: cheweiNum,
	}
}

type ShopController struct{}

// GetShopByOpenId 对齐 Java getShopByOpenId（返回 List<ShopDto>，非 Result 包装）。
func (c *ShopController) GetShopByOpenId(ctx *gin.Context) {
	openID := ctx.Query("openId")
	var userShops []model.UserShopTbl
	conf.Db.Where("open_id = ?", openID).Find(&userShops)
	list := make([]shopInfoDTO, 0, len(userShops))
	for _, us := range userShops {
		var shop model.ShopTbl
		if err := conf.Db.Where("shop_id = ?", us.ShopID).First(&shop).Error; err == nil {
			list = append(list, shopToShopInfo(shop))
		}
	}
	ctx.JSON(http.StatusOK, list)
}

// GetShopInfoByShopId 对齐 Java getShopInfoByShopId（返回 ShopDto）。
func (c *ShopController) GetShopInfoByShopId(ctx *gin.Context) {
	var shop model.ShopTbl
	if err := conf.Db.Where("shop_id = ?", ctx.Query("shopId")).First(&shop).Error; err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	ctx.JSON(http.StatusOK, shopToShopInfo(shop))
}

// GetShopName 对齐 Java getShopName（返回 String）。
func (c *ShopController) GetShopName(ctx *gin.Context) {
	var shop model.ShopTbl
	if err := conf.Db.Where("shop_id = ?", ctx.Query("shopId")).First(&shop).Error; err != nil {
		ctx.JSON(http.StatusOK, "")
		return
	}
	ctx.JSON(http.StatusOK, shop.Name)
}

// SearchShop 对齐 Java searchShop（按名称模糊查询，返回 List<Shop>）。
func (c *ShopController) SearchShop(ctx *gin.Context) {
	var rows []model.ShopTbl
	conf.Db.Where("name LIKE ?", "%"+ctx.Query("str")+"%").Find(&rows)
	list := make([]shopDTO, 0, len(rows))
	for _, s := range rows {
		list = append(list, shopToDTO(s))
	}
	ctx.JSON(http.StatusOK, list)
}

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
		FixDate:   time.Now(),
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
