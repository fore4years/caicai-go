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
	"caicai-go/repository"
)

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
	if existing := repository.PrivateChargingBySpacesNum(spacesNum); existing != nil && existing.Openid != openid {
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
	product := repository.ProductByPid(req.Pid)
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
