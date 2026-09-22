package handler

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/logger"
	"caicai-go/model"
	"caicai-go/objects"
)

// ShouyeController 对应 Java WxShouyeCon 中的首页相关端点（wxShouyeSerImpl）。
type ShouyeController struct{}

// GetMessage 获取用户消息（对应 Java wxShouyeSerImpl.getMessage）。
// 注意：user_tbl 表已无 message 列，Java 的 user.getMessage() 恒为 null，
// 因此这里忠实还原为固定返回 {"message":"null"}。
func (s *ShouyeController) GetMessage(c *gin.Context) {
	openid := c.Query("openid")
	_, err := objects.UserTbl.WithContext(c.Request.Context()).Where(objects.UserTbl.Openid.Eq(openid)).First()
	if err != nil {
		// Java 中 user 为 null 时会 NPE（500），这里降级返回空 message，避免崩溃。
		c.String(http.StatusOK, `{"message":"null"}`)
		return
	}
	c.String(http.StatusOK, `{"message":"null"}`)
}

// GetPlaces 获取附近车位信息（对应 Java wxShouyeSerImpl.getPlaces）。
func (s *ShouyeController) GetPlaces(c *gin.Context) {
	longitude := c.Query("longitude")
	latitude := c.Query("latitude")
	fanwei, _ := strconv.ParseFloat(c.Query("fanwei"), 64)
	lon, _ := strconv.ParseFloat(longitude, 64)
	lat, _ := strconv.ParseFloat(latitude, 64)

	// 对应 placeDao.selectnear
	var places []model.PlaceTbl
	if err := conf.Db.Raw(
		"SELECT * FROM place_tbl WHERE state = '可使用' AND longitude IS NOT NULL AND latitude IS NOT NULL "+
			"ORDER BY (POWER(MOD(ABS(longitude - ?), 360), 2) + POWER(ABS(latitude - ?), 2)) LIMIT 100",
		longitude, latitude,
	).Scan(&places).Error; err != nil {
		logger.Mylog.Err(err).Caller().Send()
		c.JSON(http.StatusOK, []PlaceDTO{})
		return
	}

	type placeWithDist struct {
		place model.PlaceTbl
		dist  float64
	}
	filtered := make([]placeWithDist, 0, len(places))
	for _, p := range places {
		// filter: tab_parking_spaces.enable == true
		spaces, err := objects.TabParkingSpace.WithContext(c.Request.Context()).Where(objects.TabParkingSpace.SpacesCode.Eq(p.Placeid)).First()
		if err != nil || !spaces.Enable {
			continue
		}
		pLon, _ := strconv.ParseFloat(p.Longitude, 64)
		pLat, _ := strconv.ParseFloat(p.Latitude, 64)
		dist := getDistance(pLon, pLat, lon, lat)
		// filter: 距离在范围内且在开放时间内
		if dist >= fanwei || !isOpen(p.OpenTime) {
			continue
		}
		filtered = append(filtered, placeWithDist{place: p, dist: dist})
	}

	// 按距离升序
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].dist < filtered[j].dist })

	result := make([]PlaceDTO, 0, len(filtered))
	for _, pd := range filtered {
		result = append(result, placeToDTO(pd.place))
	}
	c.JSON(http.StatusOK, result)
}

// Saoma 扫码获取车位信息（对应 Java wxShouyeSerImpl.saoma）。
func (s *ShouyeController) Saoma(c *gin.Context) {
	lockid := c.Query("lockid")

	var place model.PlaceTbl
	res := conf.Db.Raw(
		"SELECT p.* FROM place_tbl p JOIN lock_tbl l ON l.placeid = p.placeid WHERE l.lockid = ?",
		lockid,
	).Scan(&place)
	if res.Error != nil || res.RowsAffected == 0 {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, placeToDTO(place))
}

// SaomaForCharge 根据充电桩 nid 与 direction 获取车位详情（对应 Java wxShouyeSerImpl.saomaForCharge）。
func (s *ShouyeController) SaomaForCharge(c *gin.Context) {
	nid := c.Query("nid")
	direction := c.Query("direction")

	// selectCharge
	cm, err := objects.ChargeMessageTbl.WithContext(c.Request.Context()).Where(
		objects.ChargeMessageTbl.Nid.Eq(nid),
		objects.ChargeMessageTbl.Direction.Eq(direction),
		objects.ChargeMessageTbl.Chargeid.Neq(""),
	).First()
	if err != nil || cm.Placeid == "" {
		c.JSON(http.StatusOK, nil)
		return
	}

	// getLockidBypid
	var lockid string
	if lock, err := objects.LockTbl.WithContext(c.Request.Context()).Where(objects.LockTbl.Placeid.Eq(cm.Placeid)).First(); err == nil {
		lockid = lock.Lockid
	}

	// selectByPlaceid
	place, err := objects.PlaceTbl.WithContext(c.Request.Context()).Where(objects.PlaceTbl.Placeid.Eq(cm.Placeid)).First()
	if err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}

	dto := placeToPlaceDtoDTO(*place)
	dto.ChargeID = cm.Chargeid
	dto.LockID = lockid
	dto.Nid = nid
	dto.Direction = direction

	c.JSON(http.StatusOK, dto)
}

// HasOrder 判断用户是否有未完成订单（对应 Java wxShouyeSerImpl.hasOrder）。
func (s *ShouyeController) HasOrder(c *gin.Context) {
	openid := c.Query("openid")

	orders, err := objects.OrderTbl.WithContext(c.Request.Context()).Where(
		objects.OrderTbl.Openid.Eq(openid),
		objects.OrderTbl.State.Neq("已完成"),
		objects.OrderTbl.State.Neq("已取消"),
	).Find()
	if err != nil || len(orders) == 0 {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, orderToDTO(*orders[0]))
}

// HasChargeOrder 判断用户是否有未完成充电订单（对应 Java wxShouyeSerImpl.hasChargeOrder）。
func (s *ShouyeController) HasChargeOrder(c *gin.Context) {
	openid := c.Query("openid")

	orders, err := objects.ChargeOrderTbl.WithContext(c.Request.Context()).Where(
		objects.ChargeOrderTbl.Openid.Eq(openid),
		objects.ChargeOrderTbl.State.Neq("已完成"),
	).Find()
	if err != nil || len(orders) == 0 {
		c.JSON(http.StatusOK, nil)
		return
	}
	o := orders[0]
	c.JSON(http.StatusOK, ChargeOrderResultDTO{
		Orderid:   o.Orderid,
		Openid:    o.Openid,
		Lockid:    o.Lockid,
		State:     o.State,
		PlateNum:  o.PlateNum,
		BeginTime: o.BeginTime,
	})
}

// placeToDTO 将表模型映射为与 Java Place 序列化一致的 DTO。
func placeToDTO(p model.PlaceTbl) PlaceDTO {
	return PlaceDTO{
		Placeid:         p.Placeid,
		Ownerid:         p.Ownerid,
		Province:        p.Province,
		City:            p.City,
		District:        p.District,
		Streer:          p.Streer,
		RelatedBuilding: p.RelatedBuilding,
		Longitude:       p.Longitude,
		Latitude:        p.Latitude,
		Rate:            p.Rate,
		State:           p.State,
		OpenTime:        p.OpenTime,
		FixType:         p.FixType,
		Ischarge:        p.Ischarge,
		Iscarlock:       p.Iscarlock,
		ParkRate:        p.ParkRate,
		ImageID:         p.ImageID,
		Certificate:     p.Certificate,
		ChargingPid:     p.ChargingPid,
		LockPid:         p.LockPid,
		LedID:           p.LedID,
	}
}

// placeToPlaceDtoDTO 对应 Java BeanUtils.copyProperties(place, placeDto)，
// 只复制 Place 与 PlaceDto 的同名字段（含 ledId，不含 parkRate/park_rate）。
func placeToPlaceDtoDTO(p model.PlaceTbl) PlaceDtoDTO {
	return PlaceDtoDTO{
		Placeid:         p.Placeid,
		Ownerid:         p.Ownerid,
		Province:        p.Province,
		City:            p.City,
		District:        p.District,
		Streer:          p.Streer,
		RelatedBuilding: p.RelatedBuilding,
		Longitude:       p.Longitude,
		Latitude:        p.Latitude,
		Rate:            p.Rate,
		State:           p.State,
		OpenTime:        p.OpenTime,
		FixType:         p.FixType,
		Ischarge:        p.Ischarge,
		LedID:           p.LedID,
	}
}

// orderToDTO 将表模型映射为与 Java OrderBean 序列化一致的 DTO。
func orderToDTO(o model.OrderTbl) OrderDTO {
	return OrderDTO{
		Orderid:              o.Orderid,
		Openid:               o.Openid,
		Lockid:               o.Lockid,
		ReserveTime:          timeToLocal(o.ReserveTime),
		BeginTime:            timeToLocal(o.BeginTime),
		OverTime:             timeToLocal(o.OverTime),
		State:                o.State,
		TotalPrice:           o.TotalPrice,
		CloseTime:            timeToLocal(o.CloseTime),
		PlateNum:             o.PlateNum,
		SpacesID:             o.SpacesID,
		BookingStartTime:     timeToLocal(o.BookingStartTime),
		BookingEndTime:       timeToLocal(o.BookingEndTime),
		ConsumptionLocation:  o.ConsumptionLocation,
		ConsumptionType:      o.ConsumptionType,
		ChargingRates:        o.ChargingRates,
		ChargerPower:         atoi(o.ChargerPower),
		StartMode:            o.StartMode,
		StopMode:             o.StopMode,
		ChargingTime:         o.ChargingTime,
		SettlementTime:       timeToLocal(o.SettlementTime),
		BasicConsumption:     o.BasicConsumption,
		GiftAmount:           o.GiftAmount,
		TwiceChargeTime:      o.TwiceChargeTime,
		Pid:                  o.Pid,
		ChargingDegree:       o.ChargingDegree,
		FullTime:             timeToString(o.FullTime),
		LeaveTime:            timeToString(o.LeaveTime),
		ChargeGunPull:        o.ChargeGunPull,
		GunDisconnectTime:    timeToLocal(o.GunDisconnectTime),
		DecideAmount:         o.DecideAmount,
		NeighborRelocateTime: timeToLocal(o.NeighborRelocateTime),
		MoveCarTime:          timeToLocal(o.MoveCarTime),
		ImageID:              o.ImageID,
		WechatTransactionID:  o.WechatTransactionID,
	}
}

// atoi 将字符串解析为 int32，失败返回 0。
func atoi(s string) int32 {
	n, _ := strconv.Atoi(s)
	return int32(n)
}

// getDistance 根据经纬度计算两点间的球面距离（Haversine），返回单位 km。
// 对应 Java jingweiduUtil.getDistance。
func getDistance(longitude1, latitude1, longitude2, latitude2 float64) float64 {
	const earthRadius = 6378.137
	lat1 := latitude1 * math.Pi / 180
	lat2 := latitude2 * math.Pi / 180
	lng1 := longitude1 * math.Pi / 180
	lng2 := longitude2 * math.Pi / 180
	a := lat1 - lat2
	b := lng1 - lng2
	s := 2 * math.Asin(math.Sqrt(math.Pow(math.Sin(a/2), 2)+math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(b/2), 2)))
	return s * earthRadius
}

// isOpen 判断车位是否处于开放时间（对应 Java LockServiceImpl.isOpen）。
// openTime 取值："全天"、"关闭"，或 "HH:mm-HH:mm" 时间段。
func isOpen(openTime string) bool {
	if openTime == "全天" {
		return true
	}
	if openTime == "关闭" {
		return false
	}

	parts := strings.Split(openTime, "-")
	if len(parts) < 2 {
		return false
	}
	beginTime := parts[0] + ":00"
	overTime := parts[1] + ":00"

	now := time.Now().Format("15:04:05")
	nowTime, err1 := time.Parse("15:04:05", now)
	startTime, err2 := time.Parse("15:04:05", beginTime)
	endTime, err3 := time.Parse("15:04:05", overTime)
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	return isEffectiveDate(nowTime, startTime, endTime)
}

// isEffectiveDate 判断当前时间是否在 [start, end] 区间内（含边界），对应 Java LockServiceImpl.isEffectiveDate。
func isEffectiveDate(now, start, end time.Time) bool {
	if now.Equal(start) || now.Equal(end) {
		return true
	}
	return now.After(start) && now.Before(end)
}
