package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

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
		if o, err := repository.BookingOrderByLockid(lockid); err == nil && o.Openid == openid {
			repository.CancelYuyue(openid)
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
			orderid = repository.UpdateLockOrder(body.Lockid)
		} else {
			orderid = repository.CreateLockOrder(body.Openid, body.Lockid, orderid)
		}
		// Java：LockUtil.open(ldao.selectByid(lockid).getSn(), orderid) —— NB 通道开锁硬件下发，
		// Go 侧省略，仅保留订单与车位状态写入。
		_ = orderid
		repository.UpdatePlaceByLockid(body.Lockid, repository.OrderStateUse)
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
		repository.UpdatePlaceState(p.Placeid, "可使用")
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
			orderid = repository.UpdateLockOrder(body.Lockid)
		} else {
			orderid = repository.CreateLockOrder(body.Openid, body.Lockid, orderid)
		}
		repository.UpdatePlaceByLockid(body.Lockid, repository.OrderStateUse)

		// Java open_scan_lanya 无服务端硬件下发，返回 lockmac + orderid 供客户端蓝牙开锁。
		lockmac := ""
		if lock, err := repository.LockByLockid(body.Lockid); err == nil {
			lockmac = lock.Lockmac
		}
		result = append(result, lockmac, orderid)
	}
	ctx.JSON(http.StatusOK, ResultSuccess(result))
}
