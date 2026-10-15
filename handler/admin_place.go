package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/model"
	"caicai-go/repository"
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
	repository.Count(&model.PlaceTbl{}, &max)
	var places []model.PlaceTbl
	repository.Page(&model.PlaceTbl{}, "", pageOffset(current, size), size, &places)

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

	max, places := repository.PagePlacesByCity(city, pageOffset(current, size), size)

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
		max, places := repository.PagePlacesByCityState(city, dbState, pageOffset(current, size), size)
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
		max, places := repository.PagePlacesByState(dbState, pageOffset(current, size), size)
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
	owner, err := repository.OwnerByOwnerid(c.Request.Context(), c.Query("ownerid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, owner)
}

// GetUserPhone 任意方法 /system/getUserPhone?ownerid=（对齐 Java：以 ownerid 当 openid 查 user_tbl）
func (a *AdminPlaceController) GetUserPhone(c *gin.Context) {
	phone := repository.UserPhoneByOpenid(c.Query("ownerid"))
	c.String(http.StatusOK, phone)
}

// PlacesDel 任意方法 /system/placesDel，body {xuanzhong: [placeid...]}
func (a *AdminPlaceController) PlacesDel(c *gin.Context) {
	var req struct {
		Xuanzhong []string `json:"xuanzhong"`
	}
	if err := c.ShouldBindJSON(&req); err == nil {
		for _, id := range req.Xuanzhong {
			repository.DeletePlaceByPlaceid(id)
		}
	}
	c.String(http.StatusOK, "批量删除成功！")
}

// GetPlace 任意方法 /system/placeGet?placeid=
func (a *AdminPlaceController) GetPlace(c *gin.Context) {
	p, err := repository.PlaceByPlaceidDB(c.Query("placeid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, placeToDTO(*p))
}

// GetByPlaceId GET /system/getByPlaceId?placeid= → Result<PlaceVo>
func (a *AdminPlaceController) GetByPlaceId(c *gin.Context) {
	placeid := c.Query("placeid")
	if placeid == "" {
		c.JSON(http.StatusOK, ResultError(400, "placeid不能为空"))
		return
	}
	p, err := repository.PlaceByPlaceidDB(placeid)
	if err != nil {
		c.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	vo := PlaceVoDTO{PlaceDTO: placeToDTO(*p)}
	if u := repository.FindUserByOpenid(p.Ownerid); u != nil {
		dto := userToDTO(*u)
		vo.User = &dto
	}
	c.JSON(http.StatusOK, ResultSuccess(vo))
}

// LockGet 任意方法 /system/lockGet?placeid=
func (a *AdminPlaceController) LockGet(c *gin.Context) {
	l, err := repository.LockByPlaceid(c.Request.Context(), c.Query("placeid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, lockToDTO(*l))
}

// ChargeGet 任意方法 /system/chargeGet?placeid=
func (a *AdminPlaceController) ChargeGet(c *gin.Context) {
	ch, err := repository.ChargeByPlaceid(c.Query("placeid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, chargeDTO{Placeid: ch.Placeid, Charge: ch.Charge})
}

// Sousuo /system/place_list/sousuo
func (a *AdminPlaceController) Sousuo(c *gin.Context) {
	current, size := parsePage(c)
	tiaojian := c.Query("tiaojian")
	ownerid := c.Query("ownerid")

	search := func(field, val string) ([]model.PlaceTbl, int64) {
		return repository.SearchPlaces(field, val, pageOffset(current, size), size)
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
	rowsAffected := repository.UpdatePlaceStateByPlaceid(req.Placeid, state)
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}

// AdminParkingSpaceController 对齐 Java System.controller.ParkingSpacesCon。
type AdminParkingSpaceController struct{}

// GetAll GET /system/parkingSpaces/getAll
func (a *AdminParkingSpaceController) GetAll(c *gin.Context) {
	current, size := parsePage(c)
	name := c.Query("name")
	spacesCode := c.Query("spacesCode")
	id := c.Query("id")
	lockID := c.Query("lockId")
	chargingGunID := c.Query("chargingGunId")

	total, rows := repository.ListParkingSpaces(name, spacesCode, id, lockID, chargingGunID, pageOffset(current, size), size)

	c.JSON(http.StatusOK, ResultSuccessPage(total, rows))
}

// UpdateServiceAndFeeById POST /system/parkingSpaces/updateServiceAndFeeById
func (a *AdminParkingSpaceController) UpdateServiceAndFeeById(c *gin.Context) {
	id := c.Query("id")
	updates := map[string]interface{}{}
	if v, err := decimal.NewFromString(c.Query("serviceAmount")); err == nil && c.Query("serviceAmount") != "" {
		updates["service"] = v
	}
	if v, err := decimal.NewFromString(c.Query("serviceFee")); err == nil && c.Query("serviceFee") != "" {
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
	rowsAffected := repository.UpdateParkingSpaceByID(id, updates)
	c.JSON(http.StatusOK, ResultSuccess(rowsAffected > 0))
}

// GetMonthlyChargingStatistics GET /system/parkingSpaces/getMonthlyChargingStatistics
func (a *AdminParkingSpaceController) GetMonthlyChargingStatistics(c *gin.Context) {
	degree, fee, ok := repository.ParkingSpaceStats(c.Query("spacesCode"), false)
	if !ok {
		c.JSON(http.StatusOK, ResultSuccess(gin.H{"totalChargingDegree": 0, "totalElectricityFee": 0}))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(gin.H{"totalChargingDegree": degree, "totalElectricityFee": fee}))
}

// GetWeeklyChargingStatistics GET /system/parkingSpaces/getWeeklyChargingStatistics
func (a *AdminParkingSpaceController) GetWeeklyChargingStatistics(c *gin.Context) {
	degree, fee, ok := repository.ParkingSpaceStats(c.Query("spacesCode"), true)
	if !ok {
		c.JSON(http.StatusOK, ResultSuccess(gin.H{"totalChargingDegree": 0, "totalElectricityFee": 0}))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(gin.H{"totalChargingDegree": degree, "totalElectricityFee": fee}))
}
