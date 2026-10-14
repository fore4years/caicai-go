package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/conf"
	"caicai-go/model"
	"caicai-go/repository"
)

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
		BeginTime: timePtr(time.Now()),
		State:     repository.OrderStateConduct, // 进行中
	})
	return orderid
}

// updataLockOrder 对齐 LockServiceImpl.updataOrder：把该锁“预约中”订单更新为“进行中”。
func updataLockOrder(lockid string) string {
	var o model.OrderTbl
	if err := conf.Db.Where("lockid = ? AND state = ?", lockid, repository.OrderStateBooking).First(&o).Error; err != nil {
		return ""
	}
	conf.Db.Model(&model.OrderTbl{}).Where("orderid = ?", o.Orderid).Updates(map[string]interface{}{
		"begin_time": time.Now(),
		"state":      repository.OrderStateConduct,
	})
	return o.Orderid
}

// updataPlaceByLockid 对齐 LockServiceImpl.updataPlace：按 lockid 关联 placeid 更新车位状态。
func updataPlaceByLockid(lockid, state string) {
	p := repository.PlaceByLockid(lockid)
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
	p := repository.PlaceByLockid(lockid)
	if p == nil {
		return "no", ""
	}
	if p.State == "可使用" && isPlaceOpen(p.OpenTime) {
		return "ok", p.State
	}
	if p.State == "预约中" && isPlaceOpen(p.OpenTime) {
		var o model.OrderTbl
		if err := conf.Db.Where("lockid = ? AND state = ?", lockid, repository.OrderStateBooking).First(&o).Error; err == nil && o.Openid == openid {
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
		if placeState == repository.OrderStateBooking {
			orderid = updataLockOrder(body.Lockid)
		} else {
			orderid = createLockOrder(body.Openid, body.Lockid, orderid)
		}
		// Java：LockUtil.open(ldao.selectByid(lockid).getSn(), orderid) —— NB 通道开锁硬件下发，
		// Go 侧省略，仅保留订单与车位状态写入。
		_ = orderid
		updataPlaceByLockid(body.Lockid, repository.OrderStateUse)
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
	p := repository.PlaceByLockid(body.Lockid)
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
		if placeState == repository.OrderStateBooking {
			orderid = updataLockOrder(body.Lockid)
		} else {
			orderid = createLockOrder(body.Openid, body.Lockid, orderid)
		}
		updataPlaceByLockid(body.Lockid, repository.OrderStateUse)

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
