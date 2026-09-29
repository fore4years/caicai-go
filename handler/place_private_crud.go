package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============ DTO（驼峰，对齐 Java 序列化） ============

// privatePlaceDTO 对应 Java domain.PrivatePlace（驼峰）。
type privatePlaceDTO struct {
	ID            int32          `json:"id,omitempty"`
	Point         string         `json:"point,omitempty"`
	Place         string         `json:"place,omitempty"`
	Openid        string         `json:"openid,omitempty"`
	ChargingGunID int32          `json:"chargingGunId,omitempty"`
	SpacesCode    string         `json:"spacesCode,omitempty"`
	CreateTime    *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime    *LocalDateTime `json:"updateTime,omitempty"`
}

func privatePlaceToDTO(p model.PrivatePlaceTbl) privatePlaceDTO {
	return privatePlaceDTO{
		ID:            p.ID,
		Point:         p.Point,
		Place:         p.Place,
		Openid:        p.Openid,
		ChargingGunID: p.ChargingGunID,
		SpacesCode:    p.SpacesCode,
		CreateTime:    timeToLocal(p.CreateTime),
		UpdateTime:    timeToLocal(p.UpdateTime),
	}
}

// freeChargingUserDTO 对应 Java domain.FreeChargingUser（驼峰）。
type freeChargingUserDTO struct {
	ID                int32          `json:"id,omitempty"`
	Openid            string         `json:"openid,omitempty"`
	Name              string         `json:"name,omitempty"`
	Phone             string         `json:"phone,omitempty"`
	Email             string         `json:"email,omitempty"`
	ChargingStationID int32          `json:"chargingStationId,omitempty"`
	StartDate         *LocalDateTime `json:"startDate,omitempty"`
	EndDate           *LocalDateTime `json:"endDate,omitempty"`
	Status            int            `json:"status,omitempty"`
	CreateTime        *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime        *LocalDateTime `json:"updateTime,omitempty"`
	Reviser           string         `json:"reviser,omitempty"`
}

func freeChargingUserToDTO(u model.FreeChargingUsersTbl) freeChargingUserDTO {
	status := 0
	if u.Status {
		status = 1
	}
	return freeChargingUserDTO{
		ID:                u.ID,
		Openid:            u.Openid,
		Name:              u.Name,
		Phone:             u.Phone,
		Email:             u.Email,
		ChargingStationID: u.ChargingStationID,
		StartDate:         timeToLocal(u.StartDate),
		EndDate:           timeToLocal(u.EndDate),
		Status:            status,
		CreateTime:        timeToLocal(u.CreateTime),
		UpdateTime:        timeToLocal(u.UpdateTime),
		Reviser:           u.Reviser,
	}
}

// neighborShareUserDTO 对应 Java domain.NeighborShareUser（驼峰）。
type neighborShareUserDTO struct {
	ID             int32          `json:"id,omitempty"`
	OwnerUserID    int32          `json:"ownerUserId,omitempty"`
	Openid         string         `json:"openid,omitempty"`
	ChargingPileID int32          `json:"chargingPileId,omitempty"`
	Status         string         `json:"status,omitempty"`
	Phone          string         `json:"phone,omitempty"`
	CreateTime     *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime     *LocalDateTime `json:"updateTime,omitempty"`
}

func neighborShareUserToDTO(u model.NeighborShareUserTbl) neighborShareUserDTO {
	return neighborShareUserDTO{
		ID:             u.ID,
		OwnerUserID:    u.OwnerUserID,
		Openid:         u.Openid,
		ChargingPileID: u.ChargingPileID,
		Status:         u.Status,
		Phone:          u.Phone,
		CreateTime:     timeToLocal(u.CreateTime),
		UpdateTime:     timeToLocal(u.UpdateTime),
	}
}

// neighborShareDtoDTO 对应 Java dto.NeighborShareDto（neighbor + user_tbl.name/nick_name）。
type neighborShareDtoDTO struct {
	ID             int32          `json:"id,omitempty"`
	OwnerUserID    int32          `json:"ownerUserId,omitempty"`
	Openid         string         `json:"openid,omitempty"`
	ChargingPileID int32          `json:"chargingPileId,omitempty"`
	Status         string         `json:"status,omitempty"`
	Phone          string         `json:"phone,omitempty"`
	CreateTime     *LocalDateTime `json:"createTime,omitempty"`
	UpdateTime     *LocalDateTime `json:"updateTime,omitempty"`
	Name           string         `json:"name,omitempty"`
	NickName       string         `json:"nickName,omitempty"`
}

// neighborShareRow 邻享 JOIN 查询的中间扫描结构（时间用 time.Time）。
type neighborShareRow struct {
	ID             int32     `gorm:"column:id"`
	OwnerUserID    int32     `gorm:"column:owner_user_id"`
	Openid         string    `gorm:"column:openid"`
	ChargingPileID int32     `gorm:"column:charging_pile_id"`
	Status         string    `gorm:"column:status"`
	Phone          string    `gorm:"column:phone"`
	CreateTime     time.Time `gorm:"column:create_time"`
	UpdateTime     time.Time `gorm:"column:update_time"`
	Name           string    `gorm:"column:name"`
	NickName       string    `gorm:"column:nick_name"`
}

// parseTimeLoose 宽松解析日期字符串，失败返回零值。
func parseTimeLoose(s string) time.Time {
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

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
	// 对齐 Java updatePlaceById：先确认车位存在
	var original model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", p.Placeid).First(&original).Error; err != nil {
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
	list := make([]privatePlaceDTO, 0, len(rows))
	for _, p := range rows {
		list = append(list, privatePlaceToDTO(p))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *PrivatePlaceController) GetById(ctx *gin.Context) {
	var p model.PrivatePlaceTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&p).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(privatePlaceToDTO(p)))
}

func (c *PrivatePlaceController) Save(ctx *gin.Context) {
	var req struct {
		Place     string  `json:"place"`
		Openid    string  `json:"openid"`
		Pid       string  `json:"pid"`
		Code      string  `json:"code"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "参数缺失"))
		return
	}
	if req.Pid == "" {
		ctx.JSON(http.StatusOK, ResultError(400, "参数缺失"))
		return
	}

	// 对齐 Java findByPid：按产品id查充电枪
	var gun model.TabChargingGun
	if err := conf.Db.Where("product_id = ?", req.Pid).First(&gun).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "请扫描正确的充电桩二维码"))
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		ctx.JSON(http.StatusOK, ResultError(400, "城市代码不能为空"))
		return
	}
	spacesCode := nextSpacesCode("private_place_tbl", req.Code)

	// 同一充电枪只能绑定一个私桩车位
	var existing model.PrivatePlaceTbl
	if err := conf.Db.Where("charging_gun_id = ?", gun.ID).First(&existing).Error; err == nil {
		ctx.JSON(http.StatusOK, ResultError(400, "此充电枪已被绑定,请勿重复添加!"))
		return
	}

	wkt := wktPoint(req.Longitude, req.Latitude)
	if err := conf.Db.Exec(
		"INSERT INTO private_place_tbl (point, place, openid, charging_gun_id, spaces_code) VALUES (ST_GeomFromText(?),?,?,?,?)",
		wkt, req.Place, req.Openid, gun.ID, spacesCode,
	).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "添加失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(spacesCode))
}

func (c *PrivatePlaceController) UpdateById(ctx *gin.Context) {
	var p privatePlaceDTO
	if err := ctx.ShouldBindJSON(&p); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	id := p.ID
	if id == 0 {
		if v, err := strconv.Atoi(ctx.Param("id")); err == nil {
			id = int32(v)
		}
	}
	updates := map[string]interface{}{}
	if p.Place != "" {
		updates["place"] = p.Place
	}
	if p.Openid != "" {
		updates["openid"] = p.Openid
	}
	if p.ChargingGunID != 0 {
		updates["charging_gun_id"] = p.ChargingGunID
	}
	if p.SpacesCode != "" {
		updates["spaces_code"] = p.SpacesCode
	}
	res := conf.Db.Model(&model.PrivatePlaceTbl{}).Where("id = ?", id).Updates(updates)
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
	ctx.JSON(http.StatusOK, ResultSuccess(privatePlaceToDTO(p)))
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
	orderID := ctx.Param("orderId")

	var order model.OrderTbl
	if err := conf.Db.Where("orderid = ?", orderID).First(&order).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "查无此订单!"))
		return
	}
	if order.ConsumptionType != "邻享充电" {
		ctx.JSON(http.StatusOK, ResultError(400, "此订单非邻享充电订单!"))
		return
	}

	// 对齐 Java PrivatePlaceMapper.getByOrder：order → private_place → private_charging，取桩主 openid
	var openids []string
	conf.Db.Raw(
		"SELECT private_charging_tbl.openid FROM order_tbl "+
			"LEFT JOIN private_place_tbl ON private_place_tbl.id = order_tbl.spaces_id "+
			"LEFT JOIN private_charging_tbl ON private_charging_tbl.spaces_num = private_place_tbl.spaces_code "+
			"WHERE order_tbl.orderid = ?", orderID,
	).Scan(&openids)
	if len(openids) == 0 {
		ctx.JSON(http.StatusOK, ResultError(400, "此订单无私桩!"))
		return
	}
	for _, o := range openids {
		if o == order.Openid {
			ctx.JSON(http.StatusOK, ResultSuccess(true))
			return
		}
	}
	ctx.JSON(http.StatusOK, ResultSuccess(false))
}

// ============ FreeChargingUserController（/freeChargingUsers） ============

type FreeChargingUserController struct{}

func (c *FreeChargingUserController) GetAll(ctx *gin.Context) {
	var rows []model.FreeChargingUsersTbl
	conf.Db.Find(&rows)
	list := make([]freeChargingUserDTO, 0, len(rows))
	for _, u := range rows {
		list = append(list, freeChargingUserToDTO(u))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *FreeChargingUserController) GetById(ctx *gin.Context) {
	var u model.FreeChargingUsersTbl
	if err := conf.Db.Where("id = ?", ctx.Param("id")).First(&u).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(freeChargingUserToDTO(u)))
}

func (c *FreeChargingUserController) GetByStationId(ctx *gin.Context) {
	var rows []model.FreeChargingUsersTbl
	conf.Db.Where("charging_station_id = ?", ctx.Param("stationId")).Find(&rows)
	list := make([]freeChargingUserDTO, 0, len(rows))
	for _, u := range rows {
		list = append(list, freeChargingUserToDTO(u))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

func (c *FreeChargingUserController) GetByOpenId(ctx *gin.Context) {
	var u model.FreeChargingUsersTbl
	if err := conf.Db.Where("openid = ?", ctx.Param("openid")).First(&u).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(freeChargingUserToDTO(u)))
}

func (c *FreeChargingUserController) Save(ctx *gin.Context) {
	var req struct {
		Openid            string  `json:"openid"`
		Name              string  `json:"name"`
		Phone             string  `json:"phone"`
		Email             string  `json:"email"`
		ChargingStationID int32   `json:"chargingStationId"`
		StartDate         *string `json:"startDate"`
		EndDate           *string `json:"endDate"`
		Status            int     `json:"status"`
		Reviser           string  `json:"reviser"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "添加失败"))
		return
	}

	// 对齐 Java FreeChargingUserServiceImpl.add：重复申请校验
	var existing model.FreeChargingUsersTbl
	if err := conf.Db.Where("openid = ? AND charging_station_id = ?", req.Openid, req.ChargingStationID).First(&existing).Error; err == nil {
		if !existing.Status {
			ctx.JSON(http.StatusOK, ResultError(500, "您已有申请记录，请等待审核"))
			return
		}
		ctx.JSON(http.StatusOK, ResultError(500, "您已获得该充电站的服务费免除权限"))
		return
	}

	rec := model.FreeChargingUsersTbl{
		Openid:            req.Openid,
		Name:              req.Name,
		Phone:             req.Phone,
		Email:             req.Email,
		ChargingStationID: req.ChargingStationID,
		Status:            req.Status != 0,
		Reviser:           req.Reviser,
	}
	if req.StartDate != nil {
		rec.StartDate = parseTimeLoose(*req.StartDate)
	}
	if req.EndDate != nil {
		rec.EndDate = parseTimeLoose(*req.EndDate)
	}

	if err := conf.Db.Create(&rec).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(500, "添加失败"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess("添加成功"))
}

func (c *FreeChargingUserController) Update(ctx *gin.Context) {
	var req struct {
		Openid            string  `json:"openid"`
		Name              string  `json:"name"`
		Phone             string  `json:"phone"`
		Email             string  `json:"email"`
		ChargingStationID int32   `json:"chargingStationId"`
		StartDate         *string `json:"startDate"`
		EndDate           *string `json:"endDate"`
		Status            int     `json:"status"`
		Reviser           string  `json:"reviser"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	id, _ := strconv.Atoi(ctx.Param("id"))
	updates := map[string]interface{}{}
	if req.Openid != "" {
		updates["openid"] = req.Openid
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.ChargingStationID != 0 {
		updates["charging_station_id"] = req.ChargingStationID
	}
	if req.StartDate != nil {
		updates["start_date"] = parseTimeLoose(*req.StartDate)
	}
	if req.EndDate != nil {
		updates["end_date"] = parseTimeLoose(*req.EndDate)
	}
	if req.Status != 0 {
		updates["status"] = req.Status
	}
	if req.Reviser != "" {
		updates["reviser"] = req.Reviser
	}
	res := conf.Db.Model(&model.FreeChargingUsersTbl{}).Where("id = ?", id).Updates(updates)
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
	var req neighborShareUserDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "参数缺失,请检查"))
		return
	}
	if req.OwnerUserID == 0 || strings.TrimSpace(req.Openid) == "" || strings.TrimSpace(req.Status) == "" {
		ctx.JSON(http.StatusOK, ResultError(400, "参数缺失,请检查"))
		return
	}

	// 对齐 Java：校验桩主信息存在
	var charging model.PrivateChargingTbl
	if err := conf.Db.Where("id = ?", req.OwnerUserID).First(&charging).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultError(400, "查询无此桩主信息,请检查"))
		return
	}

	// 已是同一桩主的共享用户则拒绝重复申请
	var existing []model.NeighborShareUserTbl
	conf.Db.Where("openid = ?", req.Openid).Find(&existing)
	for _, n := range existing {
		if n.OwnerUserID == req.OwnerUserID {
			ctx.JSON(http.StatusOK, ResultError(400, "已是此桩主的共享用户,请勿重复申请"))
			return
		}
	}

	rec := model.NeighborShareUserTbl{
		OwnerUserID:    req.OwnerUserID,
		Openid:         req.Openid,
		ChargingPileID: req.ChargingPileID,
		Status:         req.Status,
		Phone:          req.Phone,
	}
	if err := conf.Db.Create(&rec).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(true))
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
	ctx.JSON(http.StatusOK, ResultSuccess(neighborShareUserToDTO(u)))
}

// neighborShareList 对齐 Java NeighborShareUserMapper（LEFT JOIN user_tbl 取 name/nick_name）。
func neighborShareList(ownerID, status string) []neighborShareDtoDTO {
	var rows []neighborShareRow
	db := conf.Db.Table("neighbor_share_user_tbl AS neighbor").
		Select("neighbor.id, neighbor.owner_user_id, neighbor.openid, neighbor.charging_pile_id, neighbor.status, neighbor.phone, neighbor.create_time, neighbor.update_time, user_tbl.name, user_tbl.nick_name").
		Joins("LEFT JOIN user_tbl ON neighbor.openid = user_tbl.openid").
		Where("neighbor.owner_user_id = ?", ownerID)
	if status != "" {
		db = db.Where("neighbor.status = ?", status)
	}
	db.Scan(&rows)

	list := make([]neighborShareDtoDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, neighborShareDtoDTO{
			ID:             r.ID,
			OwnerUserID:    r.OwnerUserID,
			Openid:         r.Openid,
			ChargingPileID: r.ChargingPileID,
			Status:         r.Status,
			Phone:          r.Phone,
			CreateTime:     timeToLocal(r.CreateTime),
			UpdateTime:     timeToLocal(r.UpdateTime),
			Name:           r.Name,
			NickName:       r.NickName,
		})
	}
	return list
}

func (c *NeighborShareUserController) ListByOwner(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(neighborShareList(ctx.Param("ownerId"), "")))
}

func (c *NeighborShareUserController) GetNoSharelistByOwner(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(neighborShareList(ctx.Param("ownerId"), "0")))
}

func (c *NeighborShareUserController) GetSharelistByOwner(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(neighborShareList(ctx.Param("ownerId"), "1")))
}

func (c *NeighborShareUserController) GetByPidAndOpenId(ctx *gin.Context) {
	openid := ctx.GetString("openid")
	var u model.NeighborShareUserTbl
	if err := conf.Db.Raw(
		"SELECT nsut.* FROM private_charging_tbl "+
			"LEFT JOIN private_place_tbl pp ON private_charging_tbl.spaces_num = pp.spaces_code "+
			"LEFT JOIN tab_charging_gun tcg ON pp.charging_gun_id = tcg.id "+
			"LEFT JOIN neighbor_share_user_tbl nsut ON private_charging_tbl.id = nsut.owner_user_id "+
			"WHERE tcg.product_id = ? AND nsut.openid = ?",
		ctx.Query("pid"), openid,
	).Scan(&u).Error; err != nil || u.ID == 0 {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(neighborShareUserToDTO(u)))
}
