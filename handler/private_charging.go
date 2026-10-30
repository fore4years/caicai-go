package handler

import (
	"caicai-go/logger"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

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
	imageID, err := repository.InsertImage(data)
	if err != nil {
		return 0, false
	}
	return imageID, true
}

// ============ PrivateChargingController（/privateCharging） ============

// privateShareChargingDTO 对应 Java domain.vo.PrivateShareChargingVo。
type privateShareChargingDTO struct {
	CommunityName      string          `json:"communityName,omitempty"`
	CommunitySpacesNum string          `json:"communitySpacesNum,omitempty"`
	ProductType        string          `json:"productType,omitempty"`
	ElectricityType    string          `json:"electricityType,omitempty"`
	ShareTime          string          `json:"shareTime,omitempty"`
	Fee                decimal.Decimal `json:"fee,omitempty"`
	DistanceKm         float64         `json:"distanceKm,omitempty"`
	ID                 int32           `json:"id,omitempty"`
	ImageID            int32           `json:"imageId,omitempty"`
	ExchangePlace      bool            `json:"exchangePlace,omitempty"`
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

	if _, err := repository.PrivatePlaceBySpacesCode(spacesNum); err != nil {
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

	if beanExisting, err := repository.PrivateChargingByOpenidSpacesNum(openid, spacesNum); err == nil {
		if vo.IsUpdateImg == "true" {
			imageID, ok := saveImageFromCtx(ctx)
			if !ok {
				ctx.JSON(http.StatusOK, ResultSuccess("图片上传失败"))
				return
			}
			bean.ImageID = imageID
			if beanExisting.ImageID != 0 {
				repository.DeleteImageByID(beanExisting.ImageID)
			}
		} else {
			bean.ImageID = beanExisting.ImageID
		}
		repository.UpdatePrivateChargingByOpenidSpacesNum(openid, spacesNum, bean)
	} else {
		if vo.IsUpdateImg == "true" {
			imageID, ok := saveImageFromCtx(ctx)
			if !ok {
				ctx.JSON(http.StatusOK, ResultSuccess("图片上传失败"))
				return
			}
			bean.ImageID = imageID
		}
		if err := repository.InsertPrivateCharging(&bean); err != nil {
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
		gw := repository.GatewayByMac(req.Imei)
		if gw == nil {
			gw = &model.TabGateway{Mac: req.Imei, InstallPlace: "私桩"}
			repository.CreateGateway(gw)
			if g := repository.GatewayByMac(req.Imei); g != nil {
				gw = g
			}
		}
		p.GatewayID = gw.ID
		repository.CreateProduct(&p)

		// 创建充电枪
		gun, _ := repository.ChargingGunByProductID(req.Pid)
		if gun == nil {
			gun = &model.TabChargingGun{Direction: "D", MeterValue: 0, ProductID: req.Pid}
			err := repository.CreateChargingGun(gun)
			if err != nil {
				logger.Mylog.Err(err).Msg("充电枪注册失败")
				return
			}
			gun, _ = repository.ChargingGunByProductID(req.Pid)
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
		var gunID int32
		if gun, _ := repository.ChargingGunByProductID(req.Pid); gun != nil {
			gunID = gun.ID
		}
		if pp, _ := repository.PrivatePlaceByChargingGunID(gunID); pp != nil {
			spacesNum = pp.SpacesCode
		}
	}
	rec.SpacesNum = spacesNum

	if repository.PrivateChargingBySpacesNum(spacesNum) != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "此车位编号已被人注册"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(repository.InsertPrivateCharging(&rec) == nil))
}

// insertPrivatePlace 对齐 Java PrivatePlaceServiceImpl.insert，返回生成的车位编号。
func insertPrivatePlace(req privateRegisterReq, gunID int32) string {
	if req.Code == "" {
		return ""
	}
	spacesCode := repository.NextPrivateSpacesCode(req.Code)
	if spacesCode == "" {
		return ""
	}
	if repository.PrivatePlaceExistsByChargingGunID(gunID) {
		return ""
	}
	pointWKT := fmt.Sprintf("POINT(%f %f)", req.Longitude, req.Latitude)
	if err := repository.InsertPrivatePlace(pointWKT, req.Address, "", gunID, spacesCode); err != nil {
		return ""
	}
	return spacesCode
}

func (c *PrivateChargingController) DeleteById(ctx *gin.Context) {
	res := repository.DeletePrivateChargingByID(ctx.Param("id"))
	ctx.JSON(http.StatusOK, ResultSuccess(res > 0))
}

// GetById 对齐 Java getUserAndPlaceInfoById：私桩信息 + 车位地址/充电枪。
func (c *PrivateChargingController) GetById(ctx *gin.Context) {
	p, err := repository.PrivateChargingByID(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	dto := privateChargingDetailDTO{privateChargingDTO: privateChargingToDTO(*p)}
	if point, place, gunID, err := repository.PrivatePlacePointBySpacesCode(p.SpacesNum); err == nil {
		dto.Point = point
		dto.Place = place
		dto.ChargingGunID = gunID
	}
	ctx.JSON(http.StatusOK, ResultSuccess(dto))
}

func (c *PrivateChargingController) GetByOpenId(ctx *gin.Context) {
	p, err := repository.PrivateChargingByOpenid(ctx.GetString("openid"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(*p)))
}

func (c *PrivateChargingController) GetByOpenIdAndChargeId(ctx *gin.Context) {
	p, err := repository.PrivateChargingByOpenidID(ctx.GetString("openid"), ctx.Query("id"))
	if err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(*p)))
}

func (c *PrivateChargingController) GetListByOpenId(ctx *gin.Context) {
	rows := repository.PrivateChargingsByOpenid(ctx.GetString("openid"))
	list := make([]privateChargingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateChargingToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// GetByPidAndOpenId 对齐 Java getByPidAndOpenId：经 private_place + charging_gun 关联。
func (c *PrivateChargingController) GetByPidAndOpenId(ctx *gin.Context) {
	p, err := repository.PrivateChargingByGunProductIDOpenid(ctx.Query("pid"), ctx.GetString("openid"))
	if err != nil || p == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(*p)))
}

// GetUserInfoByPid 对齐 Java getUserInfoByPid。
func (c *PrivateChargingController) GetUserInfoByPid(ctx *gin.Context) {
	p, err := repository.PrivateChargingByGunProductID(ctx.Query("pid"))
	if err != nil || p == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(*p)))
}

// GetAll 对齐 Java getAllWithPwm：私桩信息 + pwm。
func (c *PrivateChargingController) GetAll(ctx *gin.Context) {
	rows := repository.PrivateChargingsWithPwm()
	list := make([]privateChargingPwmDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateChargingPwmDTO{privateChargingDTO: privateChargingToDTO(r.PrivateChargingTbl), Pwm: r.Pwm})
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PrivateChargingController) GetBySpacesNum(ctx *gin.Context) {
	p := repository.PrivateChargingBySpacesNum(ctx.Query("spacesNum"))
	if p == nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privateChargingToDTO(*p)))
}

func (c *PrivateChargingController) Open(ctx *gin.Context) {
	res := repository.UpdatePrivateChargingOneClickOpening(ctx.Query("id"), 1)
	ctx.JSON(http.StatusOK, ResultSuccess(res > 0))
}

func (c *PrivateChargingController) UpdateShareTime(ctx *gin.Context) {
	overtimeFee := 0
	if ctx.Query("overtimeFee") != "" {
		overtimeFee, _ = strconv.Atoi(ctx.Query("overtimeFee"))
	}
	res := repository.UpdatePrivateChargingFields(ctx.Query("id"), map[string]interface{}{
		"share_time":     ctx.Query("shareTime"),
		"exchange_place": ctx.Query("exchangePlace"),
		"overtime_fee":   overtimeFee,
	})
	ctx.JSON(http.StatusOK, ResultSuccess(res > 0))
}

func (c *PrivateChargingController) Close(ctx *gin.Context) {
	res := repository.UpdatePrivateChargingOneClickOpening(ctx.Query("id"), 0)
	ctx.JSON(http.StatusOK, ResultSuccess(res > 0))
}

func (c *PrivateChargingController) GetAllSharePlace(ctx *gin.Context) {
	rows := repository.PrivateChargingsByOneClickOpening()
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
	rows := repository.PrivateShareChargingsByLocation(latitude, longitude)
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
	rows := repository.PrivateShareChargingsByName(ctx.Query("name"))
	list := make([]privateShareChargingDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, privateShareChargingToDTO(r))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PrivateChargingController) GetSharePlaceCountByName(ctx *gin.Context) {
	n := repository.CountPrivateShareChargingsByName(ctx.Query("name"))
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
	if _, err := repository.FindOrderByOrderid(req.OrderID); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	repository.UpdateOrderNeighborRelocateTime(req.OrderID)
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
	if _, err := repository.PrivateChargingByProductID(req.ProductID); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "未找到对应的私桩信息"))
		return
	}
	res := repository.UpdatePrivateChargingByProductID(req.ProductID, map[string]interface{}{
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
	ctx.JSON(http.StatusOK, ResultSuccess(res > 0))
}
