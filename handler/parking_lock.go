package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"caicai-go/model"
	"caicai-go/repository"
)

// ============ CheweisuoController（base /cheweisuo，扫码开锁） ============

// CheweisuoController 对齐 Java yingjianbufen.controller.CheweisuoCon。
type CheweisuoController struct{}

// Kaisuo /cheweisuo/kaisuo?user_ID=&lock_ID= —— 对齐 cheweisuoSerImpl.kaisuo 扫码开锁。
func (c *CheweisuoController) Kaisuo(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ResultSuccess(c.kaisuo(ctx.Query("user_ID"), ctx.Query("lock_ID"))))
}

func (c *CheweisuoController) kaisuo(userID, lockID string) string {
	live, ok := repository.PlaceLockLive(lockID)
	if !ok {
		return "服务器未找到该锁信息"
	}
	if live == "n" {
		return "这个锁没有入网哦，扫别的锁试试吧"
	}

	on, err := repository.OnlineLockByLockID(lockID)
	if err != nil {
		return "获取锁的状态失败"
	}
	if on.Battery == "n" {
		return "这个锁电量不够了哦，扫别的锁试试吧"
	}
	if on.Online == "y" {
		return "这个锁有人在用了哦，扫别的锁试试吧"
	}

	// 用户是否有未完成订单
	if repository.HasUnfinishedOrders(userID) {
		return "您有订单正在进行中，请查看"
	}

	// 锁是否被预定（预约中且非本人 → 拒绝）
	if reservation, err := repository.BookingOrderByLockid(lockID); err == nil &&
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
		BeginTime: timePtr(time.Now()),
		State:     repository.OrderStateConduct, // 进行中
	}
	repository.Create(&order)
	repository.UpdateOnlineLockOnline(lockID, "y")
	return "开锁成功"
}
