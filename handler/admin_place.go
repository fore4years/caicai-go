package handler

import (
	"database/sql"
	"github.com/shopspring/decimal"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"caicai-go/conf"
	"caicai-go/model"
)

// PlaceVoDTO 对应 Java System.domain.PlaceVo（Place 字段 + user）。
type PlaceVoDTO struct {
	PlaceDTO
	User *UserDTO `json:"user,omitempty"`
}

// lockDTO 对应 Java model.LockBean 的序列化字段（驼峰）。
type lockDTO struct {
	Lockid      string         `json:"lockid"`
	Lockmac     string         `json:"lockmac,omitempty"`
	ProductDate *LocalDateTime `json:"productDate,omitempty"`
	Factory     string         `json:"factory,omitempty"`
	FixDate     *LocalDateTime `json:"fixDate,omitempty"`
	FixType     string         `json:"fixType,omitempty"`
	Placeid     string         `json:"placeid,omitempty"`
	State       string         `json:"state,omitempty"`
	Sn          string         `json:"sn,omitempty"`
	Inductor    string         `json:"inductor,omitempty"`
	InstallTime *LocalDateTime `json:"installTime,omitempty"`
	Name        string         `json:"name,omitempty"`
	ProductID   string         `json:"productId,omitempty"`
}

func lockToDTO(l model.LockTbl) lockDTO {
	return lockDTO{
		Lockid:      l.Lockid,
		Lockmac:     l.Lockmac,
		ProductDate: timeToLocal(l.ProductDate),
		Factory:     l.Factory,
		FixDate:     timeToLocal(l.FixDate),
		FixType:     l.FixType,
		Placeid:     l.Placeid,
		State:       l.State,
		Sn:          l.Sn,
		Inductor:    l.Inductor,
		InstallTime: timeToLocal(l.InstallTime),
		Name:        l.Name,
		ProductID:   l.ProductID,
	}
}

// chargeDTO 对应 Java domain.charge。
type chargeDTO struct {
	Placeid string `json:"placeid"`
	Charge  string `json:"charge"`
}

// parkingSpaceDTO 对应 Java model.ParkingSpacesBean（驼峰）。
// price_mode / overtime_fee 用 int（DB 值可为 0/1/2），不能用 model 里的 bool。
type parkingSpaceDTO struct {
	ID            int32           `gorm:"column:id" json:"id"`
	Point         string          `gorm:"column:point" json:"point,omitempty"`
	Place         string          `gorm:"column:place" json:"place,omitempty"`
	Name          string          `gorm:"column:name" json:"name,omitempty"`
	Openid        string          `gorm:"column:openid" json:"openid,omitempty"`
	LotID         int32           `gorm:"column:lot_id" json:"lotId,omitempty"`
	LockID        string          `gorm:"column:lock_id" json:"lockId,omitempty"`
	SpacesCode    string          `gorm:"column:spaces_code" json:"spacesCode,omitempty"`
	ChargingGunID int32           `gorm:"column:charging_gun_id" json:"chargingGunId,omitempty"`
	OpenTime      string          `gorm:"column:open_time" json:"openTime,omitempty"`
	CloseTime     string          `gorm:"column:close_time" json:"closeTime,omitempty"`
	ImageID       int32           `gorm:"column:image_id" json:"imageId,omitempty"`
	Enable        bool            `gorm:"column:enable" json:"enable"`
	Certificate   int32           `gorm:"column:certificate" json:"certificate,omitempty"`
	OvertimeFee   int             `gorm:"column:overtime_fee" json:"overtimeFee,omitempty"`
	Service       decimal.Decimal `gorm:"column:service" json:"service,omitempty"`
	ServiceFee    decimal.Decimal `gorm:"column:service_fee" json:"serviceFee,omitempty"`
	PriceMode     int             `gorm:"column:price_mode" json:"priceMode,omitempty"`
}

// AdminPlaceController 对齐 Java System.controller.PlaceListCon。
type AdminPlaceController struct{}

// placePage 返回 {max, list}（raw Map，对齐 Java 旧控制器）。
func placePage(c *gin.Context, list []PlaceDTO, max int64) {
	c.JSON(http.StatusOK, gin.H{"max": max, "list": list})
}

// AllPlace 任意方法 /system/allPlace
func (a *AdminPlaceController) AllPlace(c *gin.Context) {
	current, size := parsePage(c)
	var max int64
	conf.Db.Model(&model.PlaceTbl{}).Count(&max)

	var places []model.PlaceTbl
	conf.Db.Model(&model.PlaceTbl{}).Offset(pageOffset(current, size)).Limit(size).Find(&places)

	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	placePage(c, list, max)
}

// SearchByCity GET /system/searchByCity
func (a *AdminPlaceController) SearchByCity(c *gin.Context) {
	current, size := parsePage(c)
	city := c.Query("city")
	if city == "" {
		a.AllPlace(c)
		return
	}

	var max int64
	conf.Db.Model(&model.PlaceTbl{}).Where("city LIKE ?", "%"+city+"%").Count(&max)

	var places []model.PlaceTbl
	conf.Db.Model(&model.PlaceTbl{}).Where("city LIKE ?", "%"+city+"%").
		Offset(pageOffset(current, size)).Limit(size).Find(&places)

	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	placePage(c, list, max)
}

// FilterPlacesByState GET /system/filterPlacesByState
func (a *AdminPlaceController) FilterPlacesByState(c *gin.Context) {
	current, size := parsePage(c)
	state := c.Query("state")
	city := c.Query("city")

	dbState := ""
	switch state {
	case "approved":
		dbState = "可使用"
	case "rejected":
		dbState = "未通过"
	case "pending":
		dbState = "待审核"
	}

	// 城市为空时退化为仅按状态 / 全量
	if city != "" && dbState != "" {
		var max int64
		conf.Db.Model(&model.PlaceTbl{}).Where("city = ? AND state = ?", city, dbState).Count(&max)
		var places []model.PlaceTbl
		conf.Db.Model(&model.PlaceTbl{}).Where("city = ? AND state = ?", city, dbState).
			Offset(pageOffset(current, size)).Limit(size).Find(&places)
		list := make([]PlaceDTO, 0, len(places))
		for _, p := range places {
			list = append(list, placeToDTO(p))
		}
		placePage(c, list, max)
		return
	}
	if city != "" {
		a.SearchByCity(c)
		return
	}
	if dbState != "" {
		var max int64
		conf.Db.Model(&model.PlaceTbl{}).Where("state = ?", dbState).Count(&max)
		var places []model.PlaceTbl
		conf.Db.Model(&model.PlaceTbl{}).Where("state = ?", dbState).
			Offset(pageOffset(current, size)).Limit(size).Find(&places)
		list := make([]PlaceDTO, 0, len(places))
		for _, p := range places {
			list = append(list, placeToDTO(p))
		}
		placePage(c, list, max)
		return
	}
	a.AllPlace(c)
}

// GetOwner 任意方法 /system/getOwner?ownerid=
func (a *AdminPlaceController) GetOwner(c *gin.Context) {
	var owner model.OwnerTbl
	if err := conf.Db.Where("ownerid = ?", c.Query("ownerid")).First(&owner).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, owner)
}

// GetUserPhone 任意方法 /system/getUserPhone?ownerid=（对齐 Java：以 ownerid 当 openid 查 user_tbl）
func (a *AdminPlaceController) GetUserPhone(c *gin.Context) {
	var phone string
	conf.Db.Model(&model.UserTbl{}).Where("openid = ?", c.Query("ownerid")).Pluck("phone", &phone)
	c.String(http.StatusOK, phone)
}

// PlacesDel 任意方法 /system/placesDel，body {xuanzhong: [placeid...]}
func (a *AdminPlaceController) PlacesDel(c *gin.Context) {
	var req struct {
		Xuanzhong []string `json:"xuanzhong"`
	}
	if err := c.ShouldBindJSON(&req); err == nil {
		for _, id := range req.Xuanzhong {
			conf.Db.Where("placeid = ?", id).Delete(&model.PlaceTbl{})
		}
	}
	c.String(http.StatusOK, "批量删除成功！")
}

// GetPlace 任意方法 /system/placeGet?placeid=
func (a *AdminPlaceController) GetPlace(c *gin.Context) {
	var p model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", c.Query("placeid")).First(&p).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, placeToDTO(p))
}

// GetByPlaceId GET /system/getByPlaceId?placeid= → Result<PlaceVo>
func (a *AdminPlaceController) GetByPlaceId(c *gin.Context) {
	placeid := c.Query("placeid")
	if placeid == "" {
		c.JSON(http.StatusOK, ResultError(400, "placeid不能为空"))
		return
	}
	var p model.PlaceTbl
	if err := conf.Db.Where("placeid = ?", placeid).First(&p).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	vo := PlaceVoDTO{PlaceDTO: placeToDTO(p)}
	var u model.UserTbl
	if err := conf.Db.Where("openid = ?", p.Ownerid).First(&u).Error; err == nil {
		dto := userToDTO(u)
		vo.User = &dto
	}
	c.JSON(http.StatusOK, ResultSuccess(vo))
}

// LockGet 任意方法 /system/lockGet?placeid=
func (a *AdminPlaceController) LockGet(c *gin.Context) {
	var l model.LockTbl
	if err := conf.Db.Where("placeid = ?", c.Query("placeid")).First(&l).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, lockToDTO(l))
}

// ChargeGet 任意方法 /system/chargeGet?placeid=
func (a *AdminPlaceController) ChargeGet(c *gin.Context) {
	var ch chargeDTO
	if err := conf.Db.Raw("SELECT placeid, charge FROM charge_tbl WHERE placeid = ?", c.Query("placeid")).Scan(&ch).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, ch)
}

// Sousuo /system/place_list/sousuo
func (a *AdminPlaceController) Sousuo(c *gin.Context) {
	current, size := parsePage(c)
	tiaojian := c.Query("tiaojian")
	ownerid := c.Query("ownerid")

	search := func(field, val string) ([]model.PlaceTbl, int64) {
		var max int64
		conf.Db.Model(&model.PlaceTbl{}).Where(field+" LIKE ?", "%"+val+"%").Count(&max)
		var places []model.PlaceTbl
		conf.Db.Model(&model.PlaceTbl{}).Where(field+" LIKE ?", "%"+val+"%").
			Offset(pageOffset(current, size)).Limit(size).Find(&places)
		return places, max
	}

	if ownerid != "" {
		places, max := search("ownerid", ownerid)
		if max != 0 {
			list := make([]PlaceDTO, 0, len(places))
			for _, p := range places {
				list = append(list, placeToDTO(p))
			}
			placePage(c, list, max)
			return
		}
	}

	if tiaojian != "" {
		for _, field := range []string{"placeid", "longitude", "latitude", "province", "city", "district"} {
			places, max := search(field, tiaojian)
			if max != 0 {
				list := make([]PlaceDTO, 0, len(places))
				for _, p := range places {
					list = append(list, placeToDTO(p))
				}
				placePage(c, list, max)
				return
			}
		}
	}

	placePage(c, []PlaceDTO{}, 0)
}

// UpdatePlaceStatus POST /system/updatePlaceStatus，body {placeid, enable}
func (a *AdminPlaceController) UpdatePlaceStatus(c *gin.Context) {
	var req struct {
		Placeid string `json:"placeid"`
		Enable  bool   `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Placeid == "" {
		c.JSON(http.StatusOK, ResultError(400, "placeid不能为空"))
		return
	}
	state := "未通过"
	if req.Enable {
		state = "可使用"
	}
	res := conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", req.Placeid).Update("state", state)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// AdminParkingSpaceController 对齐 Java System.controller.ParkingSpacesCon。
type AdminParkingSpaceController struct{}

// parkingSpaceQuery 构建 tab_parking_spaces 的过滤查询（返回全新链，避免被 Count/Find 污染）。
func parkingSpaceQuery(name, spacesCode, id, lockID, chargingGunID string) *gorm.DB {
	db := conf.Db.Table("tab_parking_spaces")
	if name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	if spacesCode != "" {
		db = db.Where("spaces_code LIKE ?", "%"+spacesCode+"%")
	}
	if id != "" {
		db = db.Where("id LIKE ?", "%"+id+"%")
	}
	if lockID != "" {
		db = db.Where("lock_id LIKE ?", "%"+lockID+"%")
	}
	if chargingGunID != "" {
		db = db.Where("charging_gun_id LIKE ?", "%"+chargingGunID+"%")
	}
	return db
}

// GetAll GET /system/parkingSpaces/getAll
func (a *AdminParkingSpaceController) GetAll(c *gin.Context) {
	current, size := parsePage(c)
	name := c.Query("name")
	spacesCode := c.Query("spacesCode")
	id := c.Query("id")
	lockID := c.Query("lockId")
	chargingGunID := c.Query("chargingGunId")

	var total int64
	parkingSpaceQuery(name, spacesCode, id, lockID, chargingGunID).Count(&total)

	var rows []parkingSpaceDTO
	parkingSpaceQuery(name, spacesCode, id, lockID, chargingGunID).
		Offset(pageOffset(current, size)).Limit(size).Find(&rows)

	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// UpdateServiceAndFeeById POST /system/parkingSpaces/updateServiceAndFeeById
func (a *AdminParkingSpaceController) UpdateServiceAndFeeById(c *gin.Context) {
	id := c.Query("id")
	updates := map[string]interface{}{}
	if v, err := strconv.ParseFloat(c.Query("serviceAmount"), 64); err == nil && c.Query("serviceAmount") != "" {
		updates["service"] = v
	}
	if v, err := strconv.ParseFloat(c.Query("serviceFee"), 64); err == nil && c.Query("serviceFee") != "" {
		updates["service_fee"] = v
	}
	if v, err := strconv.Atoi(c.Query("overtimeFee")); err == nil && c.Query("overtimeFee") != "" {
		updates["overtime_fee"] = v
	}
	if v, err := strconv.Atoi(c.Query("priceMode")); err == nil && c.Query("priceMode") != "" {
		updates["price_mode"] = v
	}
	if id == "" || len(updates) == 0 {
		c.JSON(http.StatusOK, ResultSuccess(false))
		return
	}
	res := conf.Db.Table("tab_parking_spaces").Where("id = ?", id).Updates(updates)
	c.JSON(http.StatusOK, ResultSuccess(res.RowsAffected > 0))
}

// currentSeason 对齐 Java SeasonUtil.getSeason()。
func currentSeason() string {
	switch int(time.Now().Month()) {
	case 3, 4, 5, 6, 10, 11:
		return "spring"
	case 7, 8, 9:
		return "summer"
	default: // 12,1,2
		return "winter"
	}
}

// parkingSpaceStats 统计某车位号下驿享充电的（月/周）电量与电费。
func parkingSpaceStats(c *gin.Context, spacesCode string, weekly bool) {
	var space model.TabParkingSpace
	if err := conf.Db.Where("spaces_code = ?", spacesCode).First(&space).Error; err != nil {
		c.JSON(http.StatusOK, ResultSuccess(gin.H{"totalChargingDegree": 0, "totalElectricityFee": 0}))
		return
	}

	now := time.Now()
	var start, end time.Time
	if weekly {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = now.AddDate(0, 0, -(weekday - 1))
	} else {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
	if weekly {
		end = start.AddDate(0, 0, 7)
	} else {
		end = start.AddDate(0, 1, 0)
	}
	startStr := start.Format("2006-01-02 15:04:05")
	endStr := end.Format("2006-01-02 15:04:05")

	// 总电量
	var degree sql.NullFloat64
	conf.Db.Raw(
		"SELECT COALESCE(SUM(charging_degree),0) FROM order_tbl WHERE spaces_id = ? AND consumption_type = '驿享充电' AND state = '已完成' AND over_time >= ? AND over_time < ?",
		space.ID, startStr, endStr,
	).Row().Scan(&degree)

	// 总电费（电子时段电价，季节表）
	season := currentSeason()
	feeSQL := "SELECT COALESCE(SUM((e.over_value - e.start_value) / 100 * " +
		"CASE s.price_mode WHEN 1 THEN p.total1 WHEN 2 THEN p.total2 ELSE p.total END),0) " +
		"FROM tab_order_electronic e " +
		"LEFT JOIN tab_price_time_" + season + " t ON t.id = e.price_time_id " +
		"LEFT JOIN tab_price p ON p.id = t.price_id " +
		"LEFT JOIN order_tbl o ON o.orderid = e.order_id " +
		"LEFT JOIN tab_parking_spaces s ON s.id = o.spaces_id " +
		"WHERE o.spaces_id = ? AND o.consumption_type = '驿享充电' AND o.state = '已完成' AND o.over_time >= ? AND o.over_time < ?"
	var fee decimal.Decimal
	conf.Db.Raw(feeSQL, space.ID, startStr, endStr).Row().Scan(&fee)

	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"totalChargingDegree": degree.Float64,
		"totalElectricityFee": fee,
	}))
}

// GetMonthlyChargingStatistics GET /system/parkingSpaces/getMonthlyChargingStatistics
func (a *AdminParkingSpaceController) GetMonthlyChargingStatistics(c *gin.Context) {
	parkingSpaceStats(c, c.Query("spacesCode"), false)
}

// GetWeeklyChargingStatistics GET /system/parkingSpaces/getWeeklyChargingStatistics
func (a *AdminParkingSpaceController) GetWeeklyChargingStatistics(c *gin.Context) {
	parkingSpaceStats(c, c.Query("spacesCode"), true)
}
