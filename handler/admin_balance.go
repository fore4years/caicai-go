package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"caicai-go/repository"
)

// AdminBalanceController 对齐 Java System.controller.BalanceDetailCon。
type AdminBalanceController struct{}

// List GET /system/balanceDetail/list
// 对齐 Java BalanceDetailSerImpl.getList：过滤 openid/phone 为空的数据，按 openid 升序。
func (a *AdminBalanceController) List(c *gin.Context) {
	current, size := parsePage(c)
	openid := c.Query("openid")
	phone := c.Query("phone")

	total, users := repository.ListUsersByBalance(openid, phone, pageOffset(current, size), size)

	list := make([]UserDTO, 0, len(users))
	for _, u := range users {
		list = append(list, userToDTO(u))
	}
	c.JSON(http.StatusOK, ResultSuccessPage(total, list))
}

// BalanceSource GET /system/balanceDetail/balanceSource?openid=
// 对齐 Java BalanceDetailSerImpl.getBalanceSource：当前余额 = 充值来源(可退) + 其他来源(不可退)。
func (a *AdminBalanceController) BalanceSource(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(http.StatusOK, ResultError(400, "用户openid不能为空"))
		return
	}
	u, err := repository.UserByOpenid(c.Request.Context(), openid)
	if err != nil {
		c.JSON(http.StatusOK, ResultError(500, "获取余额来源失败: 用户不存在"))
		return
	}
	currentBalance := u.Balans

	// 查询"已支付"且 remaining_refund_amount > 0 的充值订单，汇总为充值来源（可退）
	orders := repository.PaidRechargeOrdersByOpenid(openid)

	rechargeSource := decimal.Zero
	for _, o := range orders {
		rechargeSource = rechargeSource.Add(o.RemainingRefundAmount)
	}
	otherSource := currentBalance.Sub(rechargeSource)

	c.JSON(http.StatusOK, ResultSuccess(gin.H{
		"currentBalance":     currentBalance,
		"rechargeSource":     rechargeSource,
		"otherSource":        otherSource,
		"rechargeOrderCount": len(orders),
	}))
}

// Recharge POST /system/balanceDetail/recharge?openid=&amount=
// 对齐 Java BalanceDetailSerImpl.rechargeBalance：用户存在则 balans += amount。
// 金额用 decimal 解析并传字符串，配合 CAST(? AS DECIMAL(10,2)) 避免 MySQL DECIMAL→DOUBLE 精度丢失。
func (a *AdminBalanceController) Recharge(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(http.StatusOK, ResultError(400, "用户openid不能为空"))
		return
	}
	amount, err := decimal.NewFromString(c.Query("amount"))
	if err != nil || !amount.IsPositive() {
		c.JSON(http.StatusOK, ResultError(400, "充值金额必须大于0"))
		return
	}
	if _, err := repository.UserByOpenid(c.Request.Context(), openid); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "充值余额失败: 用户不存在"))
		return
	}
	amt := amount.StringFixed(2)
	if err := repository.RechargeBalance(openid, amt); err != nil {
		c.JSON(http.StatusOK, ResultError(500, "充值余额失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}

// Deduct POST /system/balanceDetail/deduct?openid=&amount=
// 对齐 Java BalanceDetailSerImpl.deductBalance：走统一余额消费逻辑
// （优先扣其他来源，不足部分按 FIFO 扣充值订单 remaining_refund_amount，并释放冻结余额）。
func (a *AdminBalanceController) Deduct(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(http.StatusOK, ResultError(400, "用户openid不能为空"))
		return
	}
	amount, err := decimal.NewFromString(c.Query("amount"))
	if err != nil || !amount.IsPositive() {
		c.JSON(http.StatusOK, ResultError(400, "扣除金额必须大于0"))
		return
	}
	if !repository.ConsumeBalance(openid, amount) {
		c.JSON(http.StatusOK, ResultError(500, "扣除余额失败，用户openid: "+openid+"，扣除金额: "+amount.String()))
		return
	}
	c.JSON(http.StatusOK, ResultSuccess(true))
}
