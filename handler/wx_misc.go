package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
)

// ============================================================================
// 本文件补齐 Java 侧缺失的 /wx（及 /cheweisuo、/zijian）控制器：
//   StaffController、wxGuzhangCon、CheweisuoCon、LockCon(yingjianbufen)、
//   zijianCon、owner/chargeCon。
// 硬件下发（LockUtil.open/close、车位锁开锁 HTTP 请求）无对等基础设施，均省略外部
// 下发，但保留 DB 校验与状态写入，并在省略处用注释标注。
// ============================================================================

// ============ StaffController（base /wx） ============

// StaffController 对齐 Java controller.StaffController。
type StaffController struct{}

// Selectname /wx/selectname?name= —— 对齐 StaffServiceImpl.selectname：
// 查 staff_tbl 是否存在该 name，存在返回 "ok"，否则 "fail"。
func (c *StaffController) Selectname(ctx *gin.Context) {
	name := ctx.Query("name")
	var staff model.StaffTbl
	if err := conf.Db.Where("name = ?", name).First(&staff).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess("fail"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// GetState /wx/getState?name= —— 对齐 Java getState：忽略入参，直接返回常量 "1"。
func (c *StaffController) GetState(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess("1"))
}

// ============ WxGuzhangController（base /wx，故障） ============

// WxGuzhangController 对齐 Java controller.wxGuzhangCon。
type WxGuzhangController struct{}

// GuzhangTijiao /wx/guzhangtijiao（multipart：file + fault 表单字段 + dir）
// 对齐 wxGuzhangSerImpl.submit：保存图片→生成 faultid→置 putTime/state→插入 fault_tbl。
func (c *WxGuzhangController) GuzhangTijiao(ctx *gin.Context) {
	openid := ctx.PostForm("openid")
	lockmac := ctx.PostForm("lockmac")
	errDesc := ctx.PostForm("error")

	// Java saveFileUtil.save(file, dir) 将图片上传到服务器并返回访问 URL。
	// Go 侧省略外部文件存储，仅取原始文件名占位，DB image 字段保留。
	image := ""
	if fh, e := ctx.FormFile("file"); e == nil {
		image = fh.Filename
	}

	rec := model.FaultTbl{
		Faultid: simpleUUID(),
		Openid:  openid,
		Lockmac: lockmac,
		Error:   errDesc,
		Image:   image,
		PutTime: time.Now(),
		State:   "待处理",
	}
	// Java submit 无论文件存储是否失败均返回 "ok"（IOException 被吞掉）。
	_ = conf.Db.Create(&rec)
	ctx.JSON(http.StatusOK, ResultSuccess("ok"))
}

// GetAllFault /wx/getAllfault —— 对齐 faultDao.getAll：返回全部故障。
func (c *WxGuzhangController) GetAllFault(ctx *gin.Context) {
	var rows []model.FaultTbl
	conf.Db.Find(&rows)
	list := make([]faultDTO, 0, len(rows))
	for _, f := range rows {
		list = append(list, faultToDTO(f))
	}
	ctx.JSON(http.StatusOK, ResultSuccess(list))
}

// UpdateState /wx/updateState?faultid=&state= —— 对齐 faultDao.updateState，返回受影响行数。
func (c *WxGuzhangController) UpdateState(ctx *gin.Context) {
	res := conf.Db.Model(&model.FaultTbl{}).
		Where("faultid = ?", ctx.Query("faultid")).
		Update("state", ctx.Query("state"))
	ctx.JSON(http.StatusOK, ResultSuccess(int(res.RowsAffected)))
}

// GetMaintenanceTime /wx/getMaintenance_time?Maintenance_time=&lockmac=&name=
// 对齐 faultDao.getMaintenance_time：update fault_tbl set Maintenance_time,name where lockmac。
func (c *WxGuzhangController) GetMaintenanceTime(ctx *gin.Context) {
	mt := ctx.Query("Maintenance_time")
	lockmac := ctx.Query("lockmac")
	name := ctx.Query("name")

	updates := map[string]interface{}{"name": name}
	if t := parseTimeLoose(mt); !t.IsZero() {
		updates["Maintenance_time"] = t
	}
	res := conf.Db.Model(&model.FaultTbl{}).Where("lockmac = ?", lockmac).Updates(updates)
	if res.RowsAffected > 0 {
		ctx.JSON(http.StatusOK, ResultSuccess("ok"))
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess("fail"))
}

// ============ CheweisuoController（base /cheweisuo，扫码开锁） ============

// CheweisuoController 对齐 Java yingjianbufen.controller.CheweisuoCon。
type CheweisuoController struct{}

// placeLockLive 对齐 Java place_LockDao.selectLive：select live from place_Lock_Table where lock_ID=?。
func placeLockLive(lockID string) (string, bool) {
	var live string
	if err := conf.Db.Raw("SELECT live FROM place_Lock_Table WHERE lock_ID = ?", lockID).Row().Scan(&live); err != nil {
		return "", false
	}
	return live, true
}

// Kaisuo /cheweisuo/kaisuo?user_ID=&lock_ID= —— 对齐 cheweisuoSerImpl.kaisuo 扫码开锁。
func (c *CheweisuoController) Kaisuo(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.kaisuo(ctx.Query("user_ID"), ctx.Query("lock_ID"))))
}

func (c *CheweisuoController) kaisuo(userID, lockID string) string {
	live, ok := placeLockLive(lockID)
	if !ok {
		return "服务器未找到该锁信息"
	}
	if live == "n" {
		return "这个锁没有入网哦，扫别的锁试试吧"
	}

	var on model.OnlineLockstatusTable
	if err := conf.Db.Where("lockID = ?", lockID).First(&on).Error; err != nil {
		return "获取锁的状态失败"
	}
	if on.Battery == "n" {
		return "这个锁电量不够了哦，扫别的锁试试吧"
	}
	if on.Online == "y" {
		return "这个锁有人在用了哦，扫别的锁试试吧"
	}

	// 用户是否有未完成订单
	var unfinished []model.OrderTbl
	conf.Db.Where("openid = ? AND state <> ? AND state <> ?", userID, orderStateFinished, orderStateRevoked).Find(&unfinished)
	if len(unfinished) > 0 {
		return "您有订单正在进行中，请查看"
	}

	// 锁是否被预定（预约中且非本人 → 拒绝）
	var reservation model.OrderTbl
	if err := conf.Db.Where("lockid = ? AND state = ?", lockID, orderStateBooking).First(&reservation).Error; err == nil &&
		reservation.Openid != userID {
		return "车位锁已被预定"
	}

	// 注意：Java 源码此处误写为递归调用 kaisuo 自身，且将 user_ID 与 o.getOrderid() 比较
	// （应为 o.getOpenid()）；实际意图是进入 dakai 开锁流程。Go 侧按意图还原。
	return c.dakai(userID, lockID)
}

// dakai 对齐 cheweisuoSerImpl.dakai：创建“进行中”订单 + 锁 online 置 y。
// Java 通过 qingqiu HTTP 请求车位锁开锁 url（硬件下发），此处省略。
func (c *CheweisuoController) dakai(userID, lockID string) string {
	order := model.OrderTbl{
		Orderid:   simpleUUID(),
		Openid:    userID,
		Lockid:    lockID,
		BeginTime: time.Now(),
		State:     orderStateConduct, // 进行中
	}
	conf.Db.Create(&order)
	conf.Db.Model(&model.OnlineLockstatusTable{}).Where("lockID = ?", lockID).Update("online", "y")
	return "开锁成功"
}

// ============ HardwareLockController（base /wx，LockCon yingjianbufen） ============

// HardwareLockController 对齐 Java yingjianbufen.controller.LockCon。
type HardwareLockController struct{}

// lockScanBody 对应 Java LockCon @RequestBody Map<String,String>（openid/lockid）。
type lockScanBody struct {
	Openid string `json:"openid"`
	Lockid string `json:"lockid"`
}

// isPlaceOpen 对齐 LockServiceImpl.isOpen/isEffectiveDate：
// openTime 为 "全天"/"关闭"/"HH:mm-HH:mm"，判断当前时间是否在区间内（含边界）。
func isPlaceOpen(openTime string) bool {
	if openTime == "全天" {
		return true
	}
	if openTime == "关闭" {
		return false
	}
	parts := strings.SplitN(openTime, "-", 2)
	if len(parts) != 2 {
		return false
	}
	begin := parts[0] + ":00"
	over := parts[1] + ":00"
	nowTime, e1 := time.Parse("15:04:05", time.Now().Format("15:04:05"))
	startTime, e2 := time.Parse("15:04:05", begin)
	endTime, e3 := time.Parse("15:04:05", over)
	if e1 != nil || e2 != nil || e3 != nil {
		return false
	}
	return (nowTime.Equal(startTime) || nowTime.Equal(endTime)) || (nowTime.After(startTime) && nowTime.Before(endTime))
}

// createLockOrder 对齐 LockServiceImpl.createOrder：插入“进行中”订单。
func createLockOrder(openid, lockid, orderid string) string {
	conf.Db.Create(&model.OrderTbl{
		Orderid:   orderid,
		Openid:    openid,
		Lockid:    lockid,
		BeginTime: time.Now(),
		State:     orderStateConduct, // 进行中
	})
	return orderid
}

// updataLockOrder 对齐 LockServiceImpl.updataOrder：把该锁“预约中”订单更新为“进行中”。
func updataLockOrder(lockid string) string {
	var o model.OrderTbl
	if err := conf.Db.Where("lockid = ? AND state = ?", lockid, orderStateBooking).First(&o).Error; err != nil {
		return ""
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", o.Orderid).Updates(map[string]interface{}{
		"begin_time": time.Now(),
		"state":      orderStateConduct,
	})
	return o.Orderid
}

// updataPlaceByLockid 对齐 LockServiceImpl.updataPlace：按 lockid 关联 placeid 更新车位状态。
func updataPlaceByLockid(lockid, state string) {
	p := placeByLockid(lockid)
	if p != nil {
		conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", p.Placeid).Update("state", state)
	}
}

// cancelYuyue 对齐 Java CarDetailsImpl.cancelyuyue(openid, "", "", "") 的 DB 效果。
// msg 为空 → 跳过消息；orderid/id 为空 → saveyytime.updateTime 以 id 定位，等价无操作。
func cancelYuyue(openid string) {
	// pdao.updateStateByOpenidONUsedOrder(openid, "可使用")
	conf.Db.Exec(`UPDATE place_tbl
		LEFT JOIN lock_tbl lt ON place_tbl.placeid = lt.placeid
		LEFT JOIN tab_parking_spaces ts ON lt.lockid = ts.lock_id
		LEFT JOIN order_tbl ot ON ts.id = ot.spaces_id
		SET place_tbl.state = '可使用'
		WHERE ot.openid = ? AND (ot.state = '预约中' OR ot.state = '已预约')`, openid)
	// dudao.updateReserve(openid)：reserveNumber = reserveNumber - 1
	conf.Db.Exec("UPDATE driveruse_tbl SET reserveNumber = reserveNumber - 1 WHERE openid = ?", openid)
	// odao.cancelyuyue(openid)：订单状态 → 已取消
	conf.Db.Exec("UPDATE order_tbl SET state = '已取消' WHERE openid = ? AND state = '预约中'", openid)
}

// lockIsOk 对齐 LockServiceImpl.lockIsOk：返回 (是否可用 ok/no, 车位状态)。
func lockIsOk(lockid, openid string) (string, string) {
	p := placeByLockid(lockid)
	if p == nil {
		return "no", ""
	}
	if p.State == "可使用" && isPlaceOpen(p.OpenTime) {
		return "ok", p.State
	}
	if p.State == "预约中" && isPlaceOpen(p.OpenTime) {
		var o model.OrderTbl
		if err := conf.Db.Where("lockid = ? AND state = ?", lockid, orderStateBooking).First(&o).Error; err == nil && o.Openid == openid {
			cancelYuyue(openid)
			return "ok", p.State
		}
		return "no", p.State
	}
	return "no", p.State
}

// OpenScanPhone /wx/lock/open/scan/phone —— NB 通道开锁（普通车位）。
func (c *HardwareLockController) OpenScanPhone(ctx *gin.Context) {
	var body lockScanBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess([]string{"no"}))
		return
	}
	status, placeState := lockIsOk(body.Lockid, body.Openid)
	if status == "ok" {
		orderid := simpleUUID()
		if placeState == orderStateBooking {
			orderid = updataLockOrder(body.Lockid)
		} else {
			orderid = createLockOrder(body.Openid, body.Lockid, orderid)
		}
		// Java：LockUtil.open(ldao.selectByid(lockid).getSn(), orderid) —— NB 通道开锁硬件下发，
		// Go 侧省略，仅保留订单与车位状态写入。
		_ = orderid
		updataPlaceByLockid(body.Lockid, orderStateUse)
	}
	ctx.JSON(http.StatusOK, ResultSuccess([]string{status, placeState}))
}

// CloseScanPhone /wx/lock/close/scan/phone —— NB 通道关锁（普通车位）。
func (c *HardwareLockController) CloseScanPhone(ctx *gin.Context) {
	var body lockScanBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(nil))
		return
	}
	p := placeByLockid(body.Lockid)
	if p != nil && p.State == "使用中" {
		// Java：LockUtil.close(ldao.selectByid(lockid).getSn()) —— NB 通道关锁硬件下发，此处省略。
		conf.Db.Model(&model.PlaceTbl{}).Where("placeid = ?", p.Placeid).Update("state", "可使用")
	}
	// Java close_scan_phone 返回 null（List<String>）。
	ctx.JSON(http.StatusOK, ResultSuccess(nil))
}

// OpenScanLanya /wx/lock/open/scan/lanya —— 蓝牙开锁（普通车位）。
func (c *HardwareLockController) OpenScanLanya(ctx *gin.Context) {
	var body lockScanBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess([]string{"no"}))
		return
	}
	status, placeState := lockIsOk(body.Lockid, body.Openid)
	result := []string{status, placeState}
	if status == "ok" {
		orderid := simpleUUID()
		if placeState == orderStateBooking {
			orderid = updataLockOrder(body.Lockid)
		} else {
			orderid = createLockOrder(body.Openid, body.Lockid, orderid)
		}
		updataPlaceByLockid(body.Lockid, orderStateUse)

		// Java open_scan_lanya 无服务端硬件下发，返回 lockmac + orderid 供客户端蓝牙开锁。
		var lock model.LockTbl
		lockmac := ""
		if err := conf.Db.Where("lockid = ?", body.Lockid).First(&lock).Error; err == nil {
			lockmac = lock.Lockmac
		}
		result = append(result, lockmac, orderid)
	}
	ctx.JSON(http.StatusOK, ResultSuccess(result))
}

// ============ ZijianController（base /zijian，锁自检） ============

// ZijianController 对齐 Java yingjianbufen.controller.zijianCon。
type ZijianController struct{}

// Shibai /zijian/shibai?id= —— 对齐 zijianSerImpl.shibai：自检失败，电量置 n。
func (c *ZijianController) Shibai(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.shibai(ctx.Query("id"))))
}

func (c *ZijianController) shibai(id string) int {
	var on model.OnlineLockstatusTable
	if err := conf.Db.Where("lockID = ?", id).First(&on).Error; err != nil {
		// 锁未上线，业务处理完毕
		return 1
	}
	res := conf.Db.Model(&model.OnlineLockstatusTable{}).Where("lockID = ?", id).Update("battery", "n")
	if res.RowsAffected > 0 {
		return 1
	}
	return 0
}

// Chenggong /zijian/chenggong?id= —— 对齐 zijianSerImpl.chenggong：自检成功，注册上线/更新电量。
func (c *ZijianController) Chenggong(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.chenggong(ctx.Query("id"))))
}

func (c *ZijianController) chenggong(id string) int {
	var on model.OnlineLockstatusTable
	if err := conf.Db.Where("lockID = ?", id).First(&on).Error; err != nil {
		// 锁未上线：生成 9 位编号并注册上线（battery=y, online=n）
		number := 0
		var maxNumStr string
		if e := conf.Db.Raw("SELECT number FROM online_lockstatus_Table ORDER BY number DESC LIMIT 0,1").Row().Scan(&maxNumStr); e == nil && maxNumStr != "" {
			if n, e2 := strconv.Atoi(maxNumStr); e2 == nil {
				number = n
			}
		}
		inNum := fmt.Sprintf("%09d", number+1)
		res := conf.Db.Exec("INSERT INTO online_lockstatus_Table (number, lockID, battery, online) VALUES (?, ?, 'y', 'n')", inNum, id)
		if res.RowsAffected > 0 {
			return 1
		}
		return 0
	}
	res := conf.Db.Model(&model.OnlineLockstatusTable{}).Where("lockID = ?", id).Update("battery", "y")
	if res.RowsAffected > 0 {
		return 1
	}
	return 0
}

// ============ OwnerChargeController（base /wx，owner/chargeCon） ============

// 对齐 Java ReturnConstantEnum 的返回码。
const (
	chargeObjectNull   = 10000 // 对象为空
	chargeRepeatCharge = -9    // 充电桩不能重复安装
	chargeRepeatPlace  = -10   // 车位不能重复安装充电桩
	chargeInstalled    = 10001 // 车位已安装充电桩
	chargeUninstalled  = 10002 // 车位未安装充电桩
)

// OwnerChargeController 对齐 Java owner.controller.chargeCon。
type OwnerChargeController struct{}

// AddCharge /wx/addCharge?placeid=&ownerid= —— 对齐 chargeAddSerImpl.addCharge。
func (c *OwnerChargeController) AddCharge(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.addCharge(ctx.Query("placeid"), ctx.Query("ownerid"))))
}

func (c *OwnerChargeController) addCharge(placeid, ownerid string) int {
	var existing model.ChargeTbl
	if err := conf.Db.Where("placeid = ?", placeid).First(&existing).Error; err == nil {
		return chargeRepeatPlace // -10
	}
	str := "您为车位：" + placeid + "提交的安装充电桩申请正在审核，请保持电话畅通，以便于我们的工作人员与您取得联系~~~"
	conf.Db.Create(&model.MessageTbl{
		Ownerid: ownerid,
		Message: str,
		Msgtime: time.Now().Format("2006-01-02 15:04:05"),
		Status:  "未读",
	})
	res := conf.Db.Create(&model.ChargeTbl{Placeid: placeid, Charge: "待审核"})
	return int(res.RowsAffected)
}

// AddChargeTest /wx/addChargeTest?placeid=&ownerid=&nid=&direction=&ammeter= —— 扫码添加充电桩。
func (c *OwnerChargeController) AddChargeTest(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.addChargeTest(
		ctx.Query("placeid"), ctx.Query("ownerid"), ctx.Query("nid"), ctx.Query("direction"), ctx.Query("ammeter"),
	)))
}

func (c *OwnerChargeController) addChargeTest(placeid, ownerid, nid, direction, ammeter string) int {
	// 业主和车位是否匹配
	var place model.PlaceTbl
	if err := conf.Db.Where("placeid = ? AND ownerid = ?", placeid, ownerid).First(&place).Error; err != nil {
		return chargeObjectNull // 10000
	}
	// 充电桩是否重复添加（nid + direction 且 chargeid 已分配）
	var cm model.ChargeMessageTbl
	if err := conf.Db.Where("nid = ? AND direction = ? AND chargeid IS NOT NULL", nid, direction).First(&cm).Error; err == nil {
		return chargeRepeatCharge // -9
	}
	// 车位是否重复添加
	var cm2 model.ChargeMessageTbl
	if err := conf.Db.Where("placeid = ?", placeid).First(&cm2).Error; err == nil {
		return chargeRepeatPlace // -10
	}
	// 生成充电桩编号（对齐 incrementAndFormatNumber：去掉前 2 位后 +1，再格式化为 SC%06d）
	chargeid := ""
	var maxChargeID string
	if err := conf.Db.Raw("SELECT MAX(chargeid) FROM charge_message_tbl WHERE LENGTH(chargeid) = 10").Row().Scan(&maxChargeID); err == nil && maxChargeID != "" {
		if len(maxChargeID) >= 2 {
			if n, e := strconv.Atoi(maxChargeID[2:]); e == nil {
				chargeid = fmt.Sprintf("SC%06d", n+1)
			}
		}
	}
	res := conf.Db.Exec(
		"INSERT INTO charge_message_tbl (chargeid, placeid, nid, direction, ammeter) VALUES (?, ?, ?, ?, ?)",
		chargeid, placeid, nid, direction, ammeter,
	)
	return int(res.RowsAffected)
}

// IsAddCharge /wx/isAddCharge?placeid= —— 对齐 chargeAddSerImpl.isAddCharge。
func (c *OwnerChargeController) IsAddCharge(ctx *gin.Context) {
	var cm model.ChargeMessageTbl
	if err := conf.Db.Where("placeid = ?", ctx.Query("placeid")).First(&cm).Error; err != nil {
		ctx.JSON(http.StatusOK, ResultSuccess(chargeUninstalled)) // 10002
		return
	}
	ctx.JSON(http.StatusOK, ResultSuccess(chargeInstalled)) // 10001
}
