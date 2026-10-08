package handler

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// 本文件补齐 Java 中挂 /wx 前缀的两个旧控制器（鉴权已放行，均为 @Controller +
// @ResponseBody，直接返回原始对象/字符串，而非 Result 包装）：
//   - CarDetailsCon   车辆预约 / 车牌（CarDetailsSer / CarDetailsImpl）
//   - ChargeListCon   充电列表 / 二维码（ChargeSer / ChargeSerImpl / Devcharge）
// ============================================================================

// ---------------------------- DTO（对齐 Java domain / dto 序列化） ----------------------------

// chargeMessageDTO 对应 Java domain.ChargeMessage（ProductID 首字母大写，与 Java 字段一致）。
type chargeMessageDTO struct {
	Chargeid  string `json:"chargeid,omitempty"`
	Placeid   string `json:"placeid,omitempty"`
	Nid       string `json:"nid,omitempty"`
	Direction string `json:"direction,omitempty"`
	Ammeter   string `json:"ammeter,omitempty"`
	ProductID string `json:"ProductID,omitempty"`
	Gateway   string `json:"gateway,omitempty"`
}

func chargeMessageToDTO(m model.ChargeMessageTbl) chargeMessageDTO {
	return chargeMessageDTO{
		Chargeid:  m.Chargeid,
		Placeid:   m.Placeid,
		Nid:       m.Nid,
		Direction: m.Direction,
		Ammeter:   m.Ammeter,
		ProductID: m.ProductID,
		Gateway:   m.Gateway,
	}
}

// placeResDTO 对应 Java dto.PlaceRes（BeanUtils.copyProperties(Place) 后额外 setLockId/setResKm）。
type placeResDTO struct {
	Placeid         string `json:"placeid,omitempty"`
	Province        string `json:"province,omitempty"`
	City            string `json:"city,omitempty"`
	District        string `json:"district,omitempty"`
	Streer          string `json:"streer,omitempty"`
	RelatedBuilding string `json:"relatedBuilding,omitempty"`
	Longitude       string `json:"longitude,omitempty"`
	Latitude        string `json:"latitude,omitempty"`
	Rate            string `json:"rate,omitempty"`
	State           string `json:"state,omitempty"`
	OpenTime        string `json:"openTime,omitempty"`
	FixType         string `json:"fixType,omitempty"`
	Ischarge        int32  `json:"ischarge,omitempty"`
	LockID          string `json:"lockId,omitempty"`
	ResKm           string `json:"resKm,omitempty"`
}

// chargeOrderReturnDTO 对应 Java dto.ChargeOrderReturnDto（字段为下划线命名，保持一致）。
type chargeOrderReturnDTO struct {
	Orderid        string `json:"orderid,omitempty"`
	Openid         string `json:"openid,omitempty"`
	Lockid         string `json:"lockid,omitempty"`
	PlateNum       string `json:"plate_num,omitempty"`
	TotalPrice     string `json:"totalPrice,omitempty"`
	TotalKwh       string `json:"totalKwh,omitempty"`
	TotalTime      string `json:"totalTime,omitempty"`
	Rate           string `json:"rate,omitempty"`
	ParkPrice      string `json:"park_price,omitempty"`
	ShouldGetPrice string `json:"should_get_price,omitempty"`
	ActualGetPrice string `json:"actual_get_price,omitempty"`
}

// userCarDTO 对应 Java domain.UserCar（openid / plateNum / state）。
type userCarDTO struct {
	Openid   string `json:"openid,omitempty"`
	PlateNum string `json:"plateNum,omitempty"`
	State    string `json:"state,omitempty"`
}

func userCarToDTO(u model.UsercarTbl) userCarDTO {
	return userCarDTO{Openid: u.Openid, PlateNum: u.PlateNum, State: u.State}
}

// lockPlaceidDTO 对应 Java placeDao.getByLockid 只 select l.placeid 的 LockBean 序列化结果。
type lockPlaceidDTO struct {
	Placeid string `json:"placeid,omitempty"`
}

// ---------------------------- 工具函数 ----------------------------

// genReserveOrderID 生成预约订单号：yyyyMMddHHmmssSSS + 4 位随机数字。
// 对齐 Java CarDetailsImpl.getNextOrderId（SimpleDateFormat("yyyyMMddHHmmssSSS") + CodeUtils.getNumberCode()）。
func genReserveOrderID() string {
	now := time.Now()
	return now.Format("20060102150405") + fmt.Sprintf("%03d", now.Nanosecond()/1e6) + fmt.Sprintf("%04d", rand.Intn(10000))
}

// genUUIDNoDash 生成 32 位小写 hex 随机串（去横线），对齐 Java UUIDUtil.getUUID()。
func genUUIDNoDash() string {
	const hexDigits = "0123456789abcdef"
	b := make([]byte, 32)
	for i := range b {
		b[i] = hexDigits[rand.Intn(16)]
	}
	return string(b)
}

// ============================================================================
// CarDetailsController（/wx，车辆预约 / 车牌）
// ============================================================================

type CarDetailsController struct{}

// Yuyue 车位预约。Java CarDetailsCon.yuyue / CarDetailsImpl.yuyue。
// 返回原始字符串：成功为订单号，失败为 "-8"/"-9"/"-10"。
func (c *CarDetailsController) Yuyue(ctx *gin.Context) {
	plateNum := ctx.Query("plate_num")
	lockid := ctx.Query("lockid")
	openid := ctx.Query("openid")

	orderid := genReserveOrderID()

	// 1. 查询用户未完成订单（order_tbl，state 既非已完成也非已取消）
	var list []model.OrderTbl
	conf.Db.Where("openid = ? AND state <> ? AND state <> ?", openid, "已完成", "已取消").Find(&list)

	// 2. 查询当前 lockid 是否有进行中订单（使用中/已预约/预约中）
	var orders []model.OrderTbl
	conf.Db.Where("lockid = ? AND state IN (?)", lockid, []string{"使用中", "已预约", "预约中"}).Find(&orders)
	if len(orders) > 0 {
		ctx.String(http.StatusOK, "-8")
		return
	}

	// 3. 查询预约用户信息（driveruse_tbl）；若日期不是今天则内存中重置为 3 次
	//    （Java 中 updateDriverUseData 被注释，此处同样只做内存判断、不落库）。
	var reserveNumber int32
	if du := driverUseByOpenid(openid); du != nil {
		today := time.Now().Format("2006-01-02")
		if du.EveryDay.Format("2006-01-02") != today {
			du.ReserveNumber = 3
			du.EveryDay = time.Now()
		}
		reserveNumber = du.ReserveNumber
	}
	// 无预约记录时 Java 会 NPE，这里降级为剩余次数 0（返回 "-9"），避免崩溃。

	if len(list) == 0 && reserveNumber > 0 {
		// 生成消息（预约成功）
		conf.Db.Create(&model.UsermessageTbl{
			Openid:  openid,
			Message: "预约成功",
			MsgTime: time.Now().Format("2006-01-02 15:04:05"),
			Status:  "未读",
		})

		now := time.Now()
		var spacesID int32
		if s := spacesByLockID(lockid); s != nil {
			spacesID = s.ID
		}
		// 生成订单（对齐 Java OrderBean.builder 的字段回填）
		order := model.OrderTbl{
			Orderid:         orderid,
			Openid:          openid,
			Lockid:          lockid,
			ReserveTime:     now,
			BeginTime:       now,
			State:           "预约中",
			SpacesID:        spacesID,
			PlateNum:        plateNum,
			ConsumptionType: "踩踩停车",
		}
		conf.Db.Select(
			"orderid", "openid", "lockid", "reserve_time", "begin_time",
			"state", "spaces_id", "plate_num", "consumption_type",
		).Create(&order)

		// 修改车位状态为预约中（Java pdao.updateByLockid）
		conf.Db.Exec(
			"UPDATE place_tbl LEFT JOIN lock_tbl ON place_tbl.placeid = lock_tbl.placeid SET place_tbl.state = ? WHERE lockid = ?",
			"预约中", lockid,
		)

		ctx.String(http.StatusOK, orderid)
		return
	} else if reserveNumber == 0 {
		ctx.String(http.StatusOK, "-9")
		return
	}
	ctx.String(http.StatusOK, "-10")
}

// Cancelyuyue 取消预约。Java CarDetailsCon.cancelyuyue / CarDetailsImpl.cancelyuyue。
func (c *CarDetailsController) Cancelyuyue(ctx *gin.Context) {
	openid := ctx.Query("openid")
	msg := ctx.Query("msg")
	id := ctx.Query("id")

	if msg == "超时取消" {
		conf.Db.Create(&model.UsermessageTbl{
			Openid:  openid,
			Message: "你预约的车位超时未开锁，已自动为你取消预约订单",
			Status:  "未读",
			MsgTime: time.Now().Format("2006-01-02 15:04:05"),
		})
	}

	// saveyytime_tbl 清空（Java sdao.updateTime(openid,null,null,null,id)）
	conf.Db.Exec("UPDATE saveyytime_tbl SET haveTime = NULL, exitTime = NULL, lockid = NULL WHERE id = ?", id)

	// 车位状态置为可使用（Java pdao.updateStateByOpenidONUsedOrder）
	conf.Db.Exec(
		`UPDATE place_tbl
		 LEFT JOIN lock_tbl lt ON place_tbl.placeid = lt.placeid
		 LEFT JOIN tab_parking_spaces ts ON lt.lockid = ts.lock_id
		 LEFT JOIN order_tbl ot ON ts.id = ot.spaces_id
		 SET place_tbl.state = '可使用'
		 WHERE ot.openid = ? AND (ot.state = '预约中' OR ot.state = '已预约')`,
		openid,
	)

	// 预约次数 -1（Java dudao.updateReserve）
	conf.Db.Exec("UPDATE driveruse_tbl SET reserveNumber = reserveNumber - 1 WHERE openid = ?", openid)

	// 订单状态改为已取消，返回受影响行数（Java odao.cancelyuyue）
	res := conf.Db.Exec("UPDATE order_tbl SET state = '已取消' WHERE openid = ? AND state = '预约中'", openid)
	ctx.String(http.StatusOK, strconv.FormatInt(res.RowsAffected, 10))
}

// Searchplate 查询用户首选车牌。Java CarDetailsCon.searchplate / userDao.searchplate。
func (c *CarDetailsController) Searchplate(ctx *gin.Context) {
	openid := ctx.Query("openid")
	var plateNum string
	if err := conf.Db.Raw("SELECT plate_num FROM user_tbl WHERE openid = ?", openid).Row().Scan(&plateNum); err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"plate_num": plateNum})
}

// SaveyyTime 记录预约退出时间。Java CarDetailsCon.saveyyTime / CarDetailsImpl.saveyyTime。
func (c *CarDetailsController) SaveyyTime(ctx *gin.Context) {
	havetime := ctx.Query("havetime")
	lockid := ctx.Query("lockid")
	id := ctx.Query("id")

	res := conf.Db.Exec(
		"UPDATE saveyytime_tbl SET haveTime = ?, exitTime = ?, lockid = ? WHERE id = ?",
		havetime, time.Now().Format("2006-01-02 15:04:05"), lockid, id,
	)
	ctx.String(http.StatusOK, strconv.FormatInt(res.RowsAffected, 10))
}

// GetyyTime 查询是否预约，并返回剩余时间。Java CarDetailsCon.getyyTime / CarDetailsImpl.getyyTime。
func (c *CarDetailsController) GetyyTime(ctx *gin.Context) {
	openid := ctx.Query("openid")
	lockid := ctx.Query("lockid")
	id := ctx.Query("id")

	var st model.SaveyytimeTbl
	if err := conf.Db.Where("openid = ? AND lockid = ? AND id = ?", openid, lockid, id).First(&st).Error; err != nil {
		ctx.JSON(http.StatusOK, gin.H{"key": "用户未预约"})
		return
	}

	if st.HaveTime != "" {
		htime, err := strconv.ParseInt(st.HaveTime, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusOK, nil)
			return
		}
		nowSec := time.Now().Unix()
		exitSec := st.ExitTime.Unix()
		res := nowSec - exitSec
		if res > htime {
			ctx.JSON(http.StatusOK, gin.H{"time": -1, "key": "用户已超时"})
		} else {
			ctx.JSON(http.StatusOK, gin.H{"time": strconv.FormatInt(htime-res, 10), "lockid": st.Lockid})
		}
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

// AddSaveyyTime 新增预约记录并返回 id。Java CarDetailsCon.addSaveyyTime / CarDetailsImpl.addSaveyyTime。
func (c *CarDetailsController) AddSaveyyTime(ctx *gin.Context) {
	openid := ctx.Query("openid")
	lockid := ctx.Query("lockid")
	uuid := genUUIDNoDash()
	conf.Db.Exec("INSERT INTO saveyytime_tbl (id, openid, lockid) VALUES (?, ?, ?)", uuid, openid, lockid)
	ctx.String(http.StatusOK, uuid)
}

// AddCarPlateNum 添加车牌。Java CarDetailsCon.addCarPlate_num / CarDetailsImpl.addCarPlate_num。
func (c *CarDetailsController) AddCarPlateNum(ctx *gin.Context) {
	openid := ctx.Query("openid")
	plateNum := ctx.Query("plate_num")

	var existing []model.UsercarTbl
	conf.Db.Where("openid = ?", openid).Find(&existing)
	if len(existing) >= 5 {
		ctx.JSON(http.StatusOK, []userCarDTO{{State: "认证数量以达到上限"}})
		return
	}

	// 重复车牌校验：Java 依赖 DB 唯一约束，插入抛异常时返回“该车辆已被注册”。
	var dup model.UsercarTbl
	if err := conf.Db.Where("plate_num = ? AND isDelete = 0", plateNum).First(&dup).Error; err == nil {
		ctx.JSON(http.StatusOK, []userCarDTO{{State: "该车辆已被注册"}})
		return
	}

	conf.Db.Create(&model.UsercarTbl{Openid: openid, PlateNum: plateNum, State: "未认证"})

	var rows []model.UsercarTbl
	conf.Db.Where("openid = ?", openid).Find(&rows)
	list := make([]userCarDTO, 0, len(rows))
	for _, u := range rows {
		list = append(list, userCarToDTO(u))
	}
	ctx.JSON(http.StatusOK, list)
}

// GetUserCar 获取用户车牌列表。Java CarDetailsCon.getUserCar / CarDetailsImpl.getUserCar。
func (c *CarDetailsController) GetUserCar(ctx *gin.Context) {
	openid := ctx.Query("openid")
	var rows []model.UsercarTbl
	conf.Db.Where("openid = ?", openid).Find(&rows)
	list := make([]userCarDTO, 0, len(rows))
	for _, u := range rows {
		list = append(list, userCarToDTO(u))
	}
	ctx.JSON(http.StatusOK, list)
}

// DelCarMsg 删除车牌。Java CarDetailsCon.delCarMsg / CarDetailsImpl.delCarMsg。
func (c *CarDetailsController) DelCarMsg(ctx *gin.Context) {
	plateNum := ctx.Query("plate_num")
	res := conf.Db.Where("plate_num = ?", plateNum).Delete(&model.UsercarTbl{})
	ctx.String(http.StatusOK, strconv.FormatInt(res.RowsAffected, 10))
}

// ============================================================================
// ChargeListController（/wx，充电列表 / 二维码）
// ============================================================================

type ChargeListController struct{}

// GetChargePlaces 获取附近可充电车位。Java ChargeListCon.getChargePlaces / ChargeSerImpl.getChargePlaces。
func (c *ChargeListController) GetChargePlaces(ctx *gin.Context) {
	longitude := ctx.Query("longitude")
	latitude := ctx.Query("latitude")
	fanwei, _ := strconv.ParseFloat(ctx.Query("fanwei"), 64)

	var places []model.PlaceTbl
	conf.Db.Raw(
		`SELECT p.* FROM place_tbl p
		 INNER JOIN (SELECT placeid FROM place_tbl WHERE state = '可使用' AND ischarge = 1) t ON p.placeid = t.placeid
		 WHERE p.longitude IS NOT NULL AND p.latitude IS NOT NULL
		 ORDER BY (POWER(MOD(ABS(longitude - ?), 360), 2) + POWER(ABS(latitude - ?), 2))
		 LIMIT 100`,
		longitude, latitude,
	).Scan(&places)

	lon, _ := strconv.ParseFloat(longitude, 64)
	lat, _ := strconv.ParseFloat(latitude, 64)

	result := make([]placeResDTO, 0)
	for _, p := range places {
		pLon, _ := strconv.ParseFloat(p.Longitude, 64)
		pLat, _ := strconv.ParseFloat(p.Latitude, 64)
		juli := getDistance(pLon, pLat, lon, lat)
		if juli >= fanwei {
			// Java 按距离升序，一旦超出范围即 break。
			break
		}
		if !isOpen(p.OpenTime) {
			continue
		}
		var lockid string
		conf.Db.Raw("SELECT lockid FROM lock_tbl WHERE placeid = ?", p.Placeid).Row().Scan(&lockid)
		result = append(result, placeResDTO{
			Placeid:         p.Placeid,
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
			LockID:          lockid,
			ResKm:           strconv.Itoa(int(math.Round(juli * 1000))),
		})
	}
	ctx.JSON(http.StatusOK, result)
}

// GetDevCarlock 车位锁设备信息。Java ChargeListCon.getDev_carlock / LockMapper.getDev_carlock。
func (c *ChargeListController) GetDevCarlock(ctx *gin.Context) {
	var rows []model.LockTbl
	conf.Db.Find(&rows)
	list := make([]lockDTO, 0, len(rows))
	for _, l := range rows {
		list = append(list, lockToDTO(l))
	}
	ctx.JSON(http.StatusOK, list)
}

// GetDevcharge 充电桩设备信息。Java ChargeListCon.getDevcharge / DevchargeImpl.selectAll。
func (c *ChargeListController) GetDevcharge(ctx *gin.Context) {
	var rows []model.ChargeMessageTbl
	conf.Db.Find(&rows)
	list := make([]chargeMessageDTO, 0, len(rows))
	for _, m := range rows {
		list = append(list, chargeMessageToDTO(m))
	}
	ctx.JSON(http.StatusOK, list)
}

// GetSearchcharge 搜索充电桩。Java ChargeListCon.getSearchcharge / ChargeMessageDao.selectCharge2。
func (c *ChargeListController) GetSearchcharge(ctx *gin.Context) {
	chargeid := ctx.Query("chargeid")
	var rows []model.ChargeMessageTbl
	conf.Db.Where("chargeid = ?", chargeid).Find(&rows)
	list := make([]chargeMessageDTO, 0, len(rows))
	for _, m := range rows {
		list = append(list, chargeMessageToDTO(m))
	}
	ctx.JSON(http.StatusOK, list)
}

// GetSearchcarlock 搜索车位锁。Java ChargeListCon.getSearchcarlock / LockMapper.getSearchcarlock。
func (c *ChargeListController) GetSearchcarlock(ctx *gin.Context) {
	lockid := ctx.Query("lockid")
	var rows []model.LockTbl
	conf.Db.Where("lockid = ?", lockid).Find(&rows)
	list := make([]lockDTO, 0, len(rows))
	for _, l := range rows {
		list = append(list, lockToDTO(l))
	}
	ctx.JSON(http.StatusOK, list)
}

// Getiscarlock 查询已安装车位锁的车位。Java ChargeListCon.getiscarlock / placeDao.getiscarlock。
func (c *ChargeListController) Getiscarlock(ctx *gin.Context) {
	var places []model.PlaceTbl
	conf.Db.Raw(
		"SELECT place_tbl.* FROM place_tbl INNER JOIN lock_tbl ON place_tbl.placeid = lock_tbl.placeid",
	).Scan(&places)
	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	ctx.JSON(http.StatusOK, list)
}

// SelectAllcharge 查询全部充电桩（车位）。Java ChargeListCon.selectAll / placeDao.getAll。
func (c *ChargeListController) SelectAllcharge(ctx *gin.Context) {
	var places []model.PlaceTbl
	conf.Db.Find(&places)
	list := make([]PlaceDTO, 0, len(places))
	for _, p := range places {
		list = append(list, placeToDTO(p))
	}
	ctx.JSON(http.StatusOK, list)
}

// GetinstallTime 更新安装时间。Java ChargeListCon.getinstallTime / LockMapper.getinstallTime。
func (c *ChargeListController) GetinstallTime(ctx *gin.Context) {
	installTime := ctx.Query("installTime")
	lockid := ctx.Query("lockid")
	name := ctx.Query("name")
	res := conf.Db.Exec("UPDATE lock_tbl SET install_time = ?, name = ? WHERE lockid = ?", installTime, name, lockid)
	if res.RowsAffected > 0 {
		ctx.String(http.StatusOK, "ok")
		return
	}
	ctx.String(http.StatusOK, "fail")
}

// Updateiscaolock 更新是否安装车位锁。Java ChargeListCon.updateiscaolock / placeDao.updateiscaolock。
func (c *ChargeListController) Updateiscaolock(ctx *gin.Context) {
	iscarlock, _ := strconv.Atoi(ctx.Query("iscarlock"))
	placeid := ctx.Query("placeid")
	res := conf.Db.Exec("UPDATE place_tbl SET iscarlock = ? WHERE placeid = ?", iscarlock, placeid)
	if res.RowsAffected > 0 {
		ctx.String(http.StatusOK, "ok")
		return
	}
	ctx.String(http.StatusOK, "fail")
}

// GetByLockid 查询故障锁的 placeid 列表。Java ChargeListCon.getByLockid / placeDao.getByLockid。
func (c *ChargeListController) GetByLockid(ctx *gin.Context) {
	var rows []struct {
		Placeid string
	}
	conf.Db.Raw("SELECT l.placeid FROM lock_tbl l JOIN fault_tbl f ON l.lockid = f.lockmac").Scan(&rows)
	list := make([]lockPlaceidDTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, lockPlaceidDTO{Placeid: r.Placeid})
	}
	ctx.JSON(http.StatusOK, list)
}

// GetpByLockid 按锁 id 查询车位。Java ChargeListCon.getpByLockid / placeDao.selectBylockid。
func (c *ChargeListController) GetpByLockid(ctx *gin.Context) {
	lockid := ctx.Query("lockid")
	if p := placeByLockid(lockid); p != nil {
		ctx.JSON(http.StatusOK, placeToDTO(*p))
		return
	}
	ctx.JSON(http.StatusOK, nil)
}

// GetProductIDBylockid 按产品 id 查询锁。Java ChargeListCon.getProductIDBylockid / LockMapper.getProductIDBylockid。
func (c *ChargeListController) GetProductIDBylockid(ctx *gin.Context) {
	productID := ctx.Query("ProductID")
	var rows []model.LockTbl
	conf.Db.Where("product_id = ?", productID).Find(&rows)
	list := make([]lockDTO, 0, len(rows))
	for _, l := range rows {
		list = append(list, lockToDTO(l))
	}
	ctx.JSON(http.StatusOK, list)
}

// GetAllByProductID 按产品 id 与方向查询充电桩。Java ChargeListCon.getAllByProductID / ChargeMessageDao.getAllByProductID。
func (c *ChargeListController) GetAllByProductID(ctx *gin.Context) {
	productID := ctx.Query("ProductID")
	direction := ctx.Query("direction")
	var rows []model.ChargeMessageTbl
	conf.Db.Where("ProductID = ? AND direction = ?", productID, direction).Find(&rows)
	list := make([]chargeMessageDTO, 0, len(rows))
	for _, m := range rows {
		list = append(list, chargeMessageToDTO(m))
	}
	ctx.JSON(http.StatusOK, list)
}

// CoverOrder 修改充电订单状态（结束计费）。Java ChargeListCon.coverOrder / ChargeSerImpl.coverOrder。
// 请求体：{"orderid":"..."}。返回 "ok" 或 null。
func (c *ChargeListController) CoverOrder(ctx *gin.Context) {
	var m map[string]string
	if err := ctx.ShouldBindJSON(&m); err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	orderid := m["orderid"]

	var chargeOrder model.ChargeOrderTbl
	if err := conf.Db.Where("orderid = ?", orderid).First(&chargeOrder).Error; err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}

	p := placeByLockid(chargeOrder.Lockid)
	if p == nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}

	beginTime, err := time.ParseInLocation("2006-01-02 15:04:05", chargeOrder.BeginTime, time.Local)
	if err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	useTime := time.Now().UnixMilli() - beginTime.UnixMilli()
	minutes := useTime / 1000 / 60
	rate, _ := strconv.ParseFloat(p.Rate, 64)

	var totaPrice float64
	if minutes%30 > 0 {
		totaPrice = float64(minutes/30+1) * rate
	} else {
		totaPrice = float64(minutes/30) * rate
	}

	// LockUtil.close(...) 硬件关锁指令，此处省略（无对等 MQTT/云指令基础设施）。

	conf.Db.Exec("UPDATE place_tbl SET state = '可使用' WHERE placeid = ?", p.Placeid)

	updates := map[string]interface{}{
		"openid":      chargeOrder.Openid,
		"lockid":      chargeOrder.Lockid,
		"reserveTime": chargeOrder.ReserveTime,
		"overTime":    time.Now().Format("2006-01-02 15:04:05"),
		"state":       "待支付",
		"end_kwh":     chargeOrder.EndKwh,
		"totalPrice":  decimal.NewFromFloat(totaPrice),
		"park_price":  chargeOrder.ParkPrice,
	}
	conf.Db.Model(&model.ChargeOrderTbl{}).Where("orderid = ?", orderid).Updates(updates)

	ctx.String(http.StatusOK, "ok")
}

// GetChargePayMsg 获取充电支付信息。Java ChargeListCon.getChargePayMsg / ChargeSerImpl.getChargePayMsg。
func (c *ChargeListController) GetChargePayMsg(ctx *gin.Context) {
	orderid := ctx.Query("orderid")

	var o model.ChargeOrderTbl
	if err := conf.Db.Where("orderid = ?", orderid).First(&o).Error; err != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}

	p := placeByLockid(o.Lockid)
	if p == nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}

	overTime, err1 := time.ParseInLocation("2006-01-02 15:04:05", o.OverTime, time.Local)
	beginTime, err2 := time.ParseInLocation("2006-01-02 15:04:05", o.BeginTime, time.Local)
	if err1 != nil || err2 != nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}

	minutes := (overTime.UnixMilli() - beginTime.UnixMilli()) / 1000 / 60
	hour := minutes / 60
	min := minutes % 60
	totalTime := fmt.Sprintf("%d小时 %d分钟", hour, min)

	totalKwh := fmt.Sprintf("%.2f", o.EndKwh-o.StartKwh)
	shouldGetPrice := o.TotalPrice.Add(o.ParkPrice).StringFixed(2)
	actualGetPrice := o.TotalPrice.Add(o.ParkPrice).Sub(o.CouponPrice).StringFixed(2)

	ctx.JSON(http.StatusOK, chargeOrderReturnDTO{
		Orderid:        o.Orderid,
		Openid:         o.Openid,
		Lockid:         o.Lockid,
		PlateNum:       o.PlateNum,
		TotalPrice:     o.TotalPrice.String(),
		TotalKwh:       totalKwh,
		TotalTime:      totalTime,
		Rate:           p.Rate,
		ParkPrice:      o.ParkPrice.String(),
		ShouldGetPrice: shouldGetPrice,
		ActualGetPrice: actualGetPrice,
	})
}

// GetPlaceByLockid 按锁 id 查询车位。Java ChargeListCon.getPlaceByLockid / ChargeSerImpl.getPlaceByLockid。
func (c *ChargeListController) GetPlaceByLockid(ctx *gin.Context) {
	lockid := ctx.Query("lockid")
	if p := placeByLockid(lockid); p != nil {
		ctx.JSON(http.StatusOK, placeToDTO(*p))
		return
	}
	ctx.JSON(http.StatusOK, nil)
}
